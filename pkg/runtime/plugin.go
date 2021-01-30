package runtime

import (
	docker "github.com/docker/docker/client"
	api "github.com/dodo-cli/dodo-core/api/v1alpha1"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-core/pkg/plugin/runtime"
	"github.com/dodo-cli/dodo-docker/pkg/client"
)

const name = "docker"

var _ runtime.ContainerRuntime = &ContainerRuntime{}

type ContainerRuntime struct {
	client *docker.Client
}

func New(client *docker.Client) *ContainerRuntime {
	return &ContainerRuntime{client: client}
}

func (c *ContainerRuntime) Type() plugin.Type {
	return runtime.Type
}

func (c *ContainerRuntime) Init() error {
	if c.client == nil {
		dockerClient, err := client.GetDockerClient()
		if err != nil {
			return err
		}

		c.client = dockerClient
	}

	return nil
}

func (p *ContainerRuntime) PluginInfo() (*api.PluginInfo, error) {
	return &api.PluginInfo{Name: name}, nil
}
