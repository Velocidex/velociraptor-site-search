package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/alecthomas/kingpin"
)

var (
	build_command = app.Command(
		"build", "Build index.")

	build_command_path = build_command.Arg(
		"path", "The top level path to build",
	).Required().String()

	build_command_output = build_command.Arg(
		"output", "The output for the index file",
	).Required().String()
)

func doBuild() {
	index, err := api.NewIndex(*build_command_output)
	kingpin.FatalIfError(err, "Creating index")

	err = filepath.Walk(*build_command_path,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
				return nil
			}

			page := api.NewPage()
			err = page.ParsePageFromFile(path)
			if err != nil {
				return err
			}

			// Skip pages with no rank
			if len(page.Tags) == 0 || page.Rank == 0 || page.Title == "" {
				return nil
			}

			err = index.Index(path, page)
			fmt.Printf("Indexed %v\n", path)
			return err
		})
	kingpin.FatalIfError(err, "Can not follow files")
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case build_command.FullCommand():
			doBuild()
		default:
			return false
		}
		return true
	})
}
