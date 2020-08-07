package main

import (
	"github.com/dodo/dodo-docker/pkg/runtime"
	"github.com/oclaussen/dodo/pkg/plugin"
)

func main() {
	plugin.ServePlugins(&runtime.ContainerRuntime{})
}
