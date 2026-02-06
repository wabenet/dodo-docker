package main

import (
	"os"

	plugin "github.com/wabenet/dodo-docker"
)

func main() {
	os.Exit(plugin.RunMe())
}
