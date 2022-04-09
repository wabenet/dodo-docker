package main

import (
	"os"

	"github.com/dodo-cli/dodo-docker/pkg/plugin"
)

func main() {
	os.Exit(plugin.RunMe())
}
