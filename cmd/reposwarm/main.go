package main

import "github.com/loki-bedlam/reposwarm-cli/internal/commands"

var version = "dev"

func main() {
	commands.Execute(version)
}
