package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/blevesearch/bleve/v2"
	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	config  *api.Config
	idx_mgr IndexManager
}

func (self *Server) Testing(w http.ResponseWriter, req *http.Request) {
	if req.URL.Query().Get("timewrap") != "" {
		self.idx_mgr.last_modified_time = "Sat, 28 Mar 2025 08:29:55 GMT"
		self.idx_mgr.last_download_time = time.Now().
			Add(-24 * 365 * time.Hour)
	}

	self.idx_mgr.houseKeepOnce(req.Context())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Done"))

}

func (self *Server) Query(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	now := time.Now()
	url_query := req.URL.Query()
	q, pres := url_query["q"]
	if !pres || len(q) == 0 {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "{}\n")
		return

	}

	query := bleve.NewQueryStringQuery(q[0])
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"title", "url", "tags", "rank", "crumbs"}
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.SortBy([]string{"-rank", "-_score"})

	start, pres := url_query["start"]
	if pres && len(start) > 0 {
		integer, err := strconv.ParseInt(start[0], 0, 64)
		if err == nil {
			searchRequest.From = int(integer)
		}
	}

	length, pres := url_query["len"]
	if pres && len(length) > 0 {
		integer, err := strconv.ParseInt(length[0], 0, 64)
		if err == nil {
			searchRequest.Size = int(integer)
		}
	}

	index, err := self.idx_mgr.Index(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Error: %v\n", err)
		return
	}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Error: %v\n", err)
		return
	}

	logger := self.config.GetLogger()
	logger.Info("Search from %v: %v (%v-%v) returned %v hits in %v",
		req.RemoteAddr, q[0],
		searchRequest.From,
		searchRequest.Size+searchRequest.From,
		searchResult.Total,
		time.Now().Sub(now).Round(time.Millisecond).String(),
	)

	serialized, err := json.MarshalIndent(searchResult, " ", " ")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Error: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(serialized)
}

func (self *Server) makeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/query", self.Query)
	if DEBUG {
		mux.HandleFunc("/test", self.Testing)
	}
	return mux
}

func (self *Server) Start(ctx context.Context) (err error) {
	if self.config.BindAddress == "" {
		self.config.BindAddress = "127.0.0.1:8085"
	}

	if self.config.Hostname == "" {
		return errors.New("Must specify a fully qualified hostname")
	}

	if self.config.DynDns != nil {
		if !strings.HasSuffix(self.config.Hostname, self.config.DynDns.ZoneName) {
			return fmt.Errorf("Hostname %v should be inside the zone %v",
				self.config.Hostname, self.config.DynDns.ZoneName)
		}

		updater, err := NewCloudflareUpdater(self.config)
		if err != nil {
			return err
		}
		go updater.StartDDClientService(ctx)
	}

	logger := self.config.GetLogger()

	self.idx_mgr.max_age = time.Duration(self.config.MaxIndexAgeSec) * time.Second
	if self.config.MaxIndexAgeSec == 0 {
		self.idx_mgr.max_age = 60 * time.Second
	}

	self.idx_mgr.index_path = self.config.IndexPath
	self.idx_mgr.index_url = self.config.IndexURL
	self.idx_mgr.config = self.config

	cache_dir := self.config.AutocertCertCache
	if cache_dir == "" {
		logger.Debug("autocert_cachedir is empty so will listen on plain HTTP: %v",
			self.config.BindAddress)
		return self.StartPlainHTTP(ctx)
	}

	st, err := os.Lstat(cache_dir)
	if err != nil {
		return fmt.Errorf("Unable to stat() autocert_cachedir %v: %v",
			cache_dir, err)
	}

	if !st.IsDir() {
		return fmt.Errorf("Autocert_cachedir %v is not a directory",
			cache_dir)
	}

	// This has to be fixed
	self.config.BindAddress = "0.0.0.0:443"

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(self.config.Hostname),
		Cache:      autocert.DirCache(cache_dir),
	}

	server := &http.Server{
		Addr:         self.config.BindAddress,
		Handler:      self.makeMux(),
		ErrorLog:     log.New(logger, "", 0),
		ReadTimeout:  500 * time.Second,
		WriteTimeout: 900 * time.Second,
		IdleTimeout:  150 * time.Second,
		TLSConfig:    certManager.TLSConfig(),
	}
	wg := &sync.WaitGroup{}

	sub_ctx, cancel := context.WithCancel(ctx)

	wg.Add(1)
	go self.idx_mgr.Start(sub_ctx, wg)

	// We must have port 80 open to serve the HTTP 01 challenge.
	go func() {
		defer cancel()

		err := http.ListenAndServe(":http", certManager.HTTPHandler(nil))
		if err != nil {
			logger := self.config.GetLogger()
			logger.Error("Failed to bind to http server: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		logger.Info("Starting server TLS on %v...", self.config.BindAddress)
		err := server.ListenAndServeTLS("", "")
		if err != nil {
			logger.Info("%v", err)
		}
	}()

	<-sub_ctx.Done()

	logger.Debug("Shutting down server")
	err = server.Shutdown(sub_ctx)
	if err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func (self *Server) StartPlainHTTP(ctx context.Context) (err error) {
	logger := self.config.GetLogger()

	server := &http.Server{
		Addr:         self.config.BindAddress,
		Handler:      self.makeMux(),
		ErrorLog:     log.New(logger, "", 0),
		ReadTimeout:  500 * time.Second,
		WriteTimeout: 900 * time.Second,
		IdleTimeout:  150 * time.Second,
	}
	wg := &sync.WaitGroup{}

	sub_ctx, cancel := context.WithCancel(ctx)

	wg.Add(1)
	go self.idx_mgr.Start(sub_ctx, wg)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		logger.Info("Starting plain HTTP server on %v...",
			self.config.BindAddress)
		err := server.ListenAndServe()
		if err != nil {
			logger.Info("%v", err)
		}
	}()

	<-sub_ctx.Done()

	logger.Info("Shutting down server")
	err = server.Shutdown(sub_ctx)
	if err != nil {
		return err
	}

	wg.Wait()
	return nil

}

func NewServer(config *api.Config) *Server {
	return &Server{
		config: config,
	}
}
