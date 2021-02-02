package plugin

import (
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-docker/pkg/runtime"
)

func RunMe() int {
	plugin.ServePlugins(&runtime.ContainerRuntime{})
	return 0
}

func IncludeMe() {
	plugin.IncludePlugins(&runtime.ContainerRuntime{})
}
