package main

import (
	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/Velocidex/velociraptor-site-search/server"
	"github.com/alecthomas/kingpin"
)

var (
	serve_command = app.Command(
		"serve", "Serve search index.")

	serve_command_config = serve_command.Arg(
		"path", "The path for the config file",
	).String()
)

func doServe() {
	config, err := api.LoadFromFile(*serve_command_config)
	kingpin.FatalIfError(err, "Loading config file")

	ctx, cancel := install_sig_handler()
	defer cancel()

	new_server := server.NewServer(config)
	err = new_server.Start(ctx)
	kingpin.FatalIfError(err, "Serving index")
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case serve_command.FullCommand():
			doServe()
		default:
			return false
		}
		return true
	})
}
