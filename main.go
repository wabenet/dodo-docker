package main

import (
	"os"

	"github.com/wabenet/dodo-docker/pkg/plugin"
)

func main() {
	os.Exit(plugin.RunMe())
}
