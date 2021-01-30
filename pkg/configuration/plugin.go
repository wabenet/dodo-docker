package configuration

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	api "github.com/dodo-cli/dodo-core/api/v1alpha1"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-core/pkg/plugin/configuration"
	"github.com/dodo-cli/dodo-docker/pkg/client"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
)

// TODO: a lot of this should be consolidated with runtime and client

const (
	name = "docker"

	envHost       = "DOCKER_HOST"
	envApiVersion = "DOCKER_API_VERSION"
	envCertPath   = "DOCKER_CERT_PATH"
	envVerify     = "DOCKER_TLS_VERIFY"
)

var _ configuration.Configuration = &Configuration{}

type Configuration struct {
	client     *docker.Client
	host       string
	apiVersion string
	certPath   string
}

func (p *Configuration) Type() plugin.Type {
	return configuration.Type
}

func (p *Configuration) Init() error {
	dockerClient, err := client.GetDockerClient()
	if err != nil {
		return err
	}

	p.client = dockerClient
	p.host = os.Getenv(envHost)
	p.apiVersion = os.Getenv(envApiVersion)
	p.certPath = os.Getenv(envCertPath)

	return nil
}

func (p *Configuration) PluginInfo() (*api.PluginInfo, error) {
	return &api.PluginInfo{Name: name}, nil
}

func (p *Configuration) GetBackdrop(alias string) (*api.Backdrop, error) {
	env := []*api.Environment{}

	if len(p.apiVersion) > 0 {
		env = append(env, &api.Environment{Key: envApiVersion, Value: p.apiVersion})
	}

	if len(p.host) > 0 {
		env = append(env, &api.Environment{Key: envHost, Value: p.host})
	}

	if len(p.certPath) > 0 {
		env = append(env,
			&api.Environment{Key: envCertPath, Value: p.certPath},
			&api.Environment{Key: envVerify, Value: "1"},
		)
	}

	return &api.Backdrop{Environment: env}, nil
}

func (p *Configuration) ListBackdrops() ([]*api.Backdrop, error) {
	return []*api.Backdrop{}, fmt.Errorf("not implemented") // TODO: implement list
}

func (p *Configuration) Provision(containerID string) error {
	if len(p.certPath) == 0 {
		return nil
	}

	caPath := filepath.Join(p.certPath, "ca.pem")
	ca, err := ioutil.ReadFile(caPath)
	if err != nil {
		return err
	}

	if err := p.uploadFile(containerID, caPath, ca); err != nil {
		return err
	}

	certPath := filepath.Join(p.certPath, "cert.pem")
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return err
	}

	if err := p.uploadFile(containerID, certPath, cert); err != nil {
		return err
	}

	keyPath := filepath.Join(p.certPath, "key.pem")
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}

	if err := p.uploadFile(containerID, keyPath, key); err != nil {
		return err
	}

	return nil
}

func (p *Configuration) uploadFile(containerID string, path string, contents []byte) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	go func() {
		if err := p.client.CopyToContainer(
			context.Background(),
			containerID,
			"/",
			reader,
			types.CopyToContainerOptions{},
		); err != nil {
			log.L().Error("could not upload file to container", "error", err)
		}
	}()

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	err := tarWriter.WriteHeader(&tar.Header{
		Name: path,
		Mode: 0644,
		Size: int64(len(contents)),
	})
	if err != nil {
		return err
	}

	_, err = tarWriter.Write(contents)

	return err
}
