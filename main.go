package main

import (
	"os"

	"github.com/dodo-cli/dodo-docker/plugin"
)

func main() {
	os.Exit(plugin.RunMe())
}
