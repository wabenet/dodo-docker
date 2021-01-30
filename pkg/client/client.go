package client

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

const (
	DefaultAPIVersion = "1.39"
)

func GetDockerClient() (*client.Client, error) {
	host := os.Getenv("DOCKER_HOST")
	version := os.Getenv("DOCKER_API_VERSION")
	certPath := os.Getenv("DOCKER_CERT_PATH")

	mutators := []client.Opt{}
	if len(version) > 0 {
		mutators = append(mutators, client.WithVersion(version))
	} else {
		mutators = append(mutators, client.WithVersion(DefaultAPIVersion))
	}

	if len(host) > 0 {
		mutators = append(mutators, client.WithHost(host))
	}

	if len(certPath) > 0 {
		mutators = append(mutators, client.WithTLSClientConfig(filepath.Join(certPath, "ca.pem"), filepath.Join(certPath, "cert.pem"), filepath.Join(certPath, "key.pem")))
	}

	return client.NewClientWithOpts(mutators...)
}
