package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/distribution/reference"
	log "github.com/hashicorp/go-hclog"
	"github.com/moby/moby/api/pkg/authconfig"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/registry"
	moby "github.com/moby/moby/client"
)

func (c *ContainerRuntime) ResolveImage(name string) (string, error) {
	log.L().Debug("trying to find image", "name", name)

	ref, err := reference.ParseAnyReference(name)
	if err != nil {
		return "", fmt.Errorf("could not parse image name: %w", err)
	}

	client, err := c.ensureClient()
	if err != nil {
		return "", err
	}

	_, err = client.ImageInspect(context.Background(), ref.String())
	if err == nil {
		log.L().Debug("found image locally", "ref", ref.String())

		return ref.String(), nil
	}

	parsed, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return "", fmt.Errorf("could not parse image name: %w", err)
	}

	if reference.IsNameOnly(parsed) {
		parsed = reference.TagNameOnly(parsed)
	}

	// TODO: what?
	configKey := reference.Domain(parsed)

	if configKey == "index.docker.io" {
		configKey = "docker.io"
	}

	if !strings.ContainsRune(reference.FamiliarName(parsed), '/') {
		configKey = "https://index.docker.io/v1/"
	}

	auth, _ := c.config.GetAuthConfig(configKey)

	encodedAuth, err := authconfig.Encode(registry.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		ServerAddress: auth.ServerAddress,
		Auth:          auth.Auth,
		IdentityToken: auth.IdentityToken,
		RegistryToken: auth.RegistryToken,
	})
	if err != nil {
		return "", fmt.Errorf("could not encode auth config: %w", err)
	}

	response, err := client.ImagePull(
		context.Background(),
		parsed.String(),
		moby.ImagePullOptions{
			RegistryAuth: encodedAuth,
		},
	)
	if err != nil {
		return "", fmt.Errorf("could not pull image: %w", err)
	}
	defer response.Close()

	if err = streamPull(response); err != nil {
		return "", fmt.Errorf("error during container stream: %w", err)
	}

	return parsed.String(), nil
}

func streamPull(result io.Reader) error {
	decoder := json.NewDecoder(result)

	for {
		var msg jsonstream.Message
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("invalid json: %w", err)
		}

		if msg.Error != nil {
			return msg.Error
		}

		if msg.Progress != nil {
			continue
		}

		if msg.Stream != "" || msg.Status != "" {
			log.L().Info("pull stream", "status", msg.Status, "stream", msg.Stream)
		}
	}

	return nil
}
