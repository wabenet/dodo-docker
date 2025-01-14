package runtime

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	core "github.com/wabenet/dodo-core/api/core/v1alpha6"
	"github.com/wabenet/dodo-core/pkg/plugin"
	"github.com/wabenet/dodo-core/pkg/plugin/runtime"
	"github.com/wabenet/dodo-docker/pkg/client"
)

const name = "docker"

var _ runtime.ContainerRuntime = &ContainerRuntime{}

type ContainerRuntime struct {
	client *docker.Client
}

func New() *ContainerRuntime {
	return &ContainerRuntime{}
}

func NewFromClient(client *docker.Client) *ContainerRuntime {
	return &ContainerRuntime{client: client}
}

func (*ContainerRuntime) Type() plugin.Type {
	return runtime.Type
}

func (c *ContainerRuntime) PluginInfo() *core.PluginInfo {
	return &core.PluginInfo{
		Name: &core.PluginName{
			Name: name,
			Type: runtime.Type.String(),
		},
	}
}

func (c *ContainerRuntime) Init() (plugin.Config, error) {
	client, err := c.ensureClient()
	if err != nil {
		return nil, err
	}

	ping, err := client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not reach docker host: %w", err)
	}

	return map[string]string{
		"client_version":  client.ClientVersion(),
		"host":            client.DaemonHost(),
		"api_version":     ping.APIVersion,
		"builder_version": fmt.Sprintf("%v", ping.BuilderVersion),
		"os_type":         ping.OSType,
		"experimental":    fmt.Sprintf("%t", ping.Experimental),
	}, nil
}

func (*ContainerRuntime) Cleanup() {}

func (c *ContainerRuntime) ensureClient() (*docker.Client, error) {
	if c.client == nil {
		dockerClient, err := client.GetDockerClient()
		if err != nil {
			return nil, fmt.Errorf("could not get docker config: %w", err)
		}

		c.client = dockerClient
	}

	return c.client, nil
}
