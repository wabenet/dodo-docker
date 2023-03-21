package plugin

import (
	"github.com/docker/docker/client"
	"github.com/wabenet/dodo-core/pkg/plugin"
	"github.com/wabenet/dodo-core/pkg/plugin/runtime"
	impl "github.com/wabenet/dodo-docker/internal/plugin/runtime"
)

func RunMe() int {
	m := plugin.Init()
	m.ServePlugins(NewContainerRuntime())

	return 0
}

func IncludeMe(m plugin.Manager) {
	m.IncludePlugins(NewContainerRuntime())
}

func NewContainerRuntime() runtime.ContainerRuntime {
	return impl.New()
}

func NewContainerRuntimeWithDockerClient(c *client.Client) runtime.ContainerRuntime {
	return impl.NewFromClient(c)
}
