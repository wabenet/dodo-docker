package main

import (
	"github.com/dodo-cli/dodo-docker/pkg/runtime"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
)

func main() {
	plugin.ServePlugins(&runtime.ContainerRuntime{})
}
