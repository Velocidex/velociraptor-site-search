package server

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Velocidex/velociraptor-site-search/api"
)

type IndexManager struct {
	mu sync.Mutex
	// The current index
	index              *api.Index
	index_path         string
	index_url          string
	last_modified_time string
	last_download_time time.Time
	max_age            time.Duration

	config *api.Config
}

func (self *IndexManager) Close() {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.index != nil {
		self.index.Close()
		self.index = nil
	}
}

func (self *IndexManager) wasModified() (bool, error) {
	req, err := http.NewRequest(http.MethodHead, self.index_url, nil)
	if err != nil {
		return false, err
	}

	logger := self.config.GetLogger()
	logger.Debug("Checking if index was modified from %v with HEAD check",
		self.last_download_time)

	req.Header.Add("If-Modified-Since", self.last_modified_time)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	_, err = io.ReadAll(res.Body)
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		last_modified_time := res.Header.Get("Last-Modified")
		logger.Info("wasModified: Index on server is newer at %v than %v, will fetch it",
			last_modified_time, self.last_modified_time)

		return true, nil
	}

	logger.Debug("wasModified: Index not modified")

	self.last_download_time = time.Now()
	return false, nil
}

func (self *IndexManager) fetchIndex() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, self.index_url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Record the last modified header for a header check.
	last_modified_time := res.Header.Get("Last-Modified")
	if last_modified_time != "" {
		self.last_modified_time = last_modified_time
	}

	return data, nil
}

func (self *IndexManager) Index(ctx context.Context) (*api.Index, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.index == nil {
		self.houseKeepOnce(ctx)
	}

	if self.index == nil {
		return nil, errors.New("Unable to open index")
	}

	return self.index, nil
}

func (self *IndexManager) unpackIndex(
	ctx context.Context,
	index_data []byte,
	output_dir string) error {

	reader := bytes.NewReader(index_data)

	zipfd, err := zip.NewReader(reader, int64(len(index_data)))
	if err != nil {
		return err
	}

	for _, file := range zipfd.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fd, err := file.Open()
		if err != nil {
			return err
		}

		output_path := filepath.Join(output_dir, file.Name)
		err = os.MkdirAll(filepath.Dir(output_path), 0700)
		if err != nil {
			return err
		}

		w, err := os.OpenFile(
			output_path,
			os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		w.Truncate(0)

		_, err = io.Copy(w, fd)
		if err != nil {
			return err
		}
		w.Close()
	}

	return nil
}

func (self *IndexManager) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		self.houseKeepOnce(ctx)

		select {
		case <-ctx.Done():
			return

		case <-time.After(10 * time.Second):
		}
	}
}

func (self *IndexManager) closeAndDelete(ctx context.Context) {
	index := self.index
	index_path := self.index_path

	// Try to close and remove the index in the background.
	go func() {
		logger := self.config.GetLogger()
		logger.Debug("Closing index at %v", index_path)

		// Try to remove the index 10 time, and if we cant, just give
		// up.
		for i := 0; i < 10; i++ {
			if index != nil {
				err := index.Purge()
				if err == nil {
					index = nil
				} else {
					logger.Debug("While purging index %v: %v",
						index_path, err)
				}
			}

			if index == nil {
				err := os.RemoveAll(index_path)
				if err == nil {
					return
				}
				logger.Debug("While removing index %v: %v",
					index_path, err)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
		}
		logger.Error("Unable to remove previous index %v", index_path)
	}()
}

func (self *IndexManager) houseKeepOnce(ctx context.Context) {
	self.mu.Lock()
	defer self.mu.Unlock()

	now := time.Now()
	logger := self.config.GetLogger()

	if self.index != nil {
		// Index is still fairly fresh nothing to do.
		logger.Debug("Checking index from %v which is %v old max age %v",
			self.last_modified_time,
			now.Sub(self.last_download_time).Round(time.Second).String(),
			self.max_age.String())

		if self.last_download_time.Add(self.max_age).After(now) {
			return
		}

		// Check if the index is older on the server
		was_mod, err := self.wasModified()
		if !was_mod && err == nil {
			return
		}

		// Close the old index so we can fetch a new one.
		self.closeAndDelete(ctx)
	}

	new_index_path, err := os.MkdirTemp("", "tmp*")
	if err != nil {
		logger.Debug("Can not create tmp directory %v", err)
		return
	}

	idx_data, err := self.fetchIndex()
	if err != nil {
		logger.Debug("Can not fetch index %v", err)
		return
	}

	logger.Info("Fetched index data of %v bytes, last modified %v",
		len(idx_data), self.last_modified_time)

	// Unpack the index into the temp directory
	err = self.unpackIndex(ctx, idx_data, new_index_path)
	if err != nil {
		logger.Error("Can not unpack index %v", err)
		return
	}

	self.index, err = api.OpenIndex(new_index_path)
	if err != nil {
		logger.Error("Can not open index %v", err)
		return
	}

	self.index_path = new_index_path
	self.last_download_time = time.Now()
	logger.Info("New index is ready at %v", new_index_path)
}
