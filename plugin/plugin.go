package plugin

import (
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-docker/pkg/plugin/runtime"
	log "github.com/hashicorp/go-hclog"
)

func RunMe() int {
	if err := plugin.ServePlugins(runtime.New()); err != nil {
		log.L().Error("error serving plugin", "error", err)
	}

	return 0
}

func IncludeMe() {
	plugin.IncludePlugins(runtime.New())
}
