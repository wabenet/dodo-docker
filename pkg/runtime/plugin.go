package runtime

import (
	docker "github.com/docker/docker/client"
	"github.com/dodo/dodo-docker/pkg/client"
	"github.com/oclaussen/dodo/pkg/plugin"
	"github.com/oclaussen/dodo/pkg/plugin/runtime"
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
