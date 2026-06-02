package runtime

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	moby "github.com/moby/moby/client"
	"github.com/wabenet/dodo-core/pkg/plugin"
	"github.com/wabenet/dodo-core/pkg/plugin/runtime"
)

const (
	name = "docker"

	defaultDockerContext = "default"
	defaultDockerHost    = "unix:///var/run/docker.sock"
)

var _ runtime.ContainerRuntime = &ContainerRuntime{}

type ContainerRuntime struct {
	client moby.APIClient
	config *configfile.ConfigFile
}

func New() *ContainerRuntime {
	return &ContainerRuntime{}
}

func NewFromClient(client moby.APIClient) *ContainerRuntime {
	return &ContainerRuntime{client: client}
}

func (*ContainerRuntime) Type() plugin.Type {
	return runtime.Type
}

func (c *ContainerRuntime) Metadata() plugin.Metadata {
	return plugin.NewMetadata(runtime.Type, name)
}

func (c *ContainerRuntime) Init() (plugin.Config, error) {
	client, err := c.ensureClient()
	if err != nil {
		return nil, err
	}

	ping, err := client.Ping(context.Background(), moby.PingOptions{})
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

func (c *ContainerRuntime) ensureClient() (moby.APIClient, error) { //nolint:ireturn
	if c.client == nil {
		c.config = config.LoadDefaultConfigFile(nil)

		endpoint, err := getDockerEndpoint(c.config)
		if err != nil {
			return nil, fmt.Errorf("could not get docker endpoint: %w", err)
		}

		opts, err := endpoint.ClientOpts()
		if err != nil {
			return nil, fmt.Errorf("could not get endpoint options: %w", err)
		}

		client, err := moby.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("could not create moby client: %w", err)
		}

		c.client = client
	}

	return c.client, nil
}

func getDockerEndpoint(configFile *configfile.ConfigFile) (docker.Endpoint, error) {
	ctxName := getContextName(configFile)

	if ctxName == defaultDockerContext {
		return defaultDockerEndpoint()
	}

	return dockerEndpointFromContext(ctxName)
}

func getContextName(configFile *configfile.ConfigFile) string {
	if os.Getenv(moby.EnvOverrideHost) != "" {
		return defaultDockerContext
	}

	if ctxName := os.Getenv("DOCKER_CONTEXT"); ctxName != "" {
		return ctxName
	}

	if configFile.CurrentContext != "" {
		return configFile.CurrentContext
	}

	return defaultDockerContext
}

func defaultDockerEndpoint() (docker.Endpoint, error) {
	endpoint := docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			Host:          defaultDockerHost,
			SkipTLSVerify: false,
		},
	}

	if override := os.Getenv(moby.EnvOverrideHost); override != "" {
		// The original Docker CLI uses a whole lot of logic to infer a valid endpoint from the env var,
		// including lots of default values and so on.
		// We are just assuming the user passes the endpoint in the correct format and hope for the best.
		endpoint.Host = override
	}

	return endpoint, nil
}

func dockerEndpointFromContext(name string) (docker.Endpoint, error) {
	ctxStore := store.New(
		config.ContextStoreDir(),
		store.NewConfig(
			func() any { return &dockerContext{} },
			[]store.NamedTypeGetter{
				store.EndpointTypeGetter(docker.DockerEndpoint, func() any { return &docker.EndpointMeta{} }),
			}...,
		),
	)

	ctxMeta, err := ctxStore.GetMetadata(name)
	if err != nil {
		return docker.Endpoint{}, fmt.Errorf("could not get context metadata: %w", err)
	}

	epMeta, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return docker.Endpoint{}, fmt.Errorf("could not get endpoint from context: %w", err)
	}

	endpoint, err := docker.WithTLSData(ctxStore, name, epMeta)
	if err != nil {
		return docker.Endpoint{}, fmt.Errorf("could not create endpoint: %w", err)
	}

	return endpoint, nil
}

type dockerContext struct {
	Description      string
	AdditionalFields map[string]any
}
