package main

import (
	"github.com/dodo/dodo-docker/pkg/runtime"
	dodo "github.com/oclaussen/dodo/pkg/plugin"
)

func main() {
	runtime.RegisterPlugin()
	dodo.ServePlugins()
}
