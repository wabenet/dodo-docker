package plugin

import (
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-docker/pkg/plugin/runtime"
)

func RunMe() int {
	plugin.ServePlugins(runtime.New())
	return 0
}

func IncludeMe() {
	plugin.IncludePlugins(runtime.New())
}
