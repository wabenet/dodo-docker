package main

import (
	"github.com/dodo/dodo-docker/pkg/runtime"
	"github.com/dodo/dodo-core/pkg/plugin"
)

func main() {
	plugin.ServePlugins(&runtime.ContainerRuntime{})
}
