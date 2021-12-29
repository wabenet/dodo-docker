package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

const (
	DefaultAPIVersion = "1.39"

	EnvConfigPath = "DODO_DOCKER_CONFIG"
	EnvAPIVersion = "DOCKER_API_VERSION"
	EnvHost       = "DOCKER_HOST"
	EnvCertPath   = "DOCKER_CERT_PATH"
)

type Config struct {
	APIVersion string
	Host       string
	CaPath     string
	CertPath   string
	KeyPath    string
}

func GetDockerClient() (*client.Client, error) {
	if configPath := os.Getenv(EnvConfigPath); configPath != "" {
		return DockerClientFromConfigFile(configPath)
	}

	return DockerClientFromEnv()
}

func DockerClientFromConfigFile(path string) (*client.Client, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': %w", path, err)
	}

	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	return DockerClientFromConfig(&config)
}

func DockerClientFromEnv() (*client.Client, error) {
	c := &Config{
		APIVersion: os.Getenv(EnvAPIVersion),
		Host:       os.Getenv(EnvHost),
	}

	if certPath := os.Getenv(EnvCertPath); certPath != "" {
		c.CaPath = filepath.Join(certPath, "ca.pem")
		c.CertPath = filepath.Join(certPath, "cert.pem")
		c.KeyPath = filepath.Join(certPath, "key.pem")
	}

	return DockerClientFromConfig(c)
}

func DockerClientFromConfig(c *Config) (*client.Client, error) {
	mutators := []client.Opt{}

	if c.APIVersion != "" {
		mutators = append(mutators, client.WithVersion(c.APIVersion))
	} else {
		mutators = append(mutators, client.WithVersion(DefaultAPIVersion))
	}

	if c.Host != "" {
		mutators = append(mutators, client.WithHost(c.Host))
	}

	if c.CaPath != "" && c.CertPath != "" && c.KeyPath != "" {
		mutators = append(mutators, client.WithTLSClientConfig(c.CaPath, c.CertPath, c.KeyPath))
	}

	return client.NewClientWithOpts(mutators...)
}
