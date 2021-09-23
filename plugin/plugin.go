package plugin

import (
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-docker/pkg/plugin/runtime"
)

func RunMe() int {
	m := plugin.Init()
	m.ServePlugins(runtime.New())

	return 0
}

func IncludeMe(m plugin.Manager) {
	m.IncludePlugins(runtime.New())
}
