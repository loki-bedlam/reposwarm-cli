package main

import "github.com/loki-bedlam/reposwarm-cli/internal/commands"

var version = "1.0.0"

func main() {
	commands.Execute(version)
}
