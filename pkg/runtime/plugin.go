package runtime

import (
	docker "github.com/docker/docker/client"
	"github.com/dodo/dodo-docker/pkg/client"
	"github.com/oclaussen/dodo/pkg/container"
	"github.com/oclaussen/dodo/pkg/plugin"
)

type ContainerRuntime struct {
	client *docker.Client
}

func RegisterPlugin() {
	plugin.RegisterPluginServer(
		container.PluginType,
		&container.Plugin{Impl: &ContainerRuntime{}},
	)
}

func (c *ContainerRuntime) Init() error {
	dockerClient, err := client.GetDockerClient()
	if err != nil {
		return err
	}

	c.client = dockerClient

	return nil
}
