package main

import (
	"fmt"

	"github.com/Velocidex/velociraptor-site-search/parser"
	"github.com/alecthomas/kingpin"
	"github.com/goccy/go-yaml"
)

var (
	parse_command = app.Command(
		"parse", "Parse a markdown page.")

	parse_command_path = parse_command.Arg(
		"path", "The top level path to build",
	).Required().String()
)

func doParse() {
	page, err := parser.ParsePageFromFile(*parse_command_path)
	kingpin.FatalIfError(err, "ParsePageFromFile")

	serialized, err := yaml.Marshal(page)
	kingpin.FatalIfError(err, "Marshal")

	fmt.Printf("Text: \n%s\n%v", page.Text, string(serialized))
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case parse_command.FullCommand():
			doParse()
		default:
			return false
		}
		return true
	})
}
