package runtime

import (
	docker "github.com/docker/docker/client"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-core/pkg/plugin/runtime"
	"github.com/dodo-cli/dodo-docker/pkg/client"
)

type ContainerRuntime struct {
	client *docker.Client
}

func (c *ContainerRuntime) Type() plugin.Type {
	return runtime.Type
}

func (c *ContainerRuntime) Init() error {
	dockerClient, err := client.GetDockerClient()
	if err != nil {
		return err
	}

	c.client = dockerClient

	return nil
}
