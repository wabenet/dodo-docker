package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/distribution/reference"
	cli "github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
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

	if _, _, err := client.ImageInspectWithRaw(context.Background(), ref.String()); err == nil {
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

	repoInfo, err := registry.ParseRepositoryInfo(parsed)
	if err != nil {
		return "", fmt.Errorf("could not parse image name: %w", err)
	}

	dockerCLI, err := cli.NewDockerCli(cli.WithBaseContext(context.Background()))
	if err != nil {
		return "", fmt.Errorf("could not get docker config: %w", err)
	}

	authConfig := cli.ResolveAuthConfig(dockerCLI.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return "", fmt.Errorf("could not encode auth config: %w", err)
	}

	response, err := client.ImagePull(
		context.Background(),
		parsed.String(),
		image.PullOptions{
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
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("invalid json: %w", err)
		}

		if msg.Error != nil {
			return msg.Error
		}

		if msg.Progress != nil || msg.ProgressMessage != "" {
			continue
		}

		if msg.Stream != "" || msg.Status != "" {
			log.L().Info("pull stream", "status", msg.Status, "stream", msg.Stream)
		}
	}

	return nil
}
