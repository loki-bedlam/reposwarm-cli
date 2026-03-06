package main

import "github.com/reposwarm/reposwarm-cli/internal/commands"

var version = "dev"

func main() {
	commands.Execute(version)
}
