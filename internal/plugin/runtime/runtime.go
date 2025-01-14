package runtime

import (
	"context"
	"fmt"

	cli "github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	docker "github.com/docker/docker/client"
	core "github.com/wabenet/dodo-core/api/core/v1alpha6"
	"github.com/wabenet/dodo-core/pkg/plugin"
	"github.com/wabenet/dodo-core/pkg/plugin/runtime"
)

const name = "docker"

var _ runtime.ContainerRuntime = &ContainerRuntime{}

type ContainerRuntime struct {
	client docker.APIClient
}

func New() *ContainerRuntime {
	return &ContainerRuntime{}
}

func NewFromClient(client docker.APIClient) *ContainerRuntime {
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

func (c *ContainerRuntime) ensureClient() (docker.APIClient, error) {
	if c.client == nil {
		dockerCLI, err := cli.NewDockerCli(cli.WithBaseContext(context.Background()))
		if err != nil {
			return nil, fmt.Errorf("could not get docker config: %w", err)
		}

		if err := dockerCLI.Initialize(&cliflags.ClientOptions{}); err != nil {
			return nil, fmt.Errorf("could not get docker config: %w", err)
		}

		c.client = dockerCLI.Client()
	}

	return c.client, nil
}
