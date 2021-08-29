package runtime

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	docker "github.com/dodo-cli/dodo-docker/pkg/client"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) ResolveImage(name string) (string, error) {
	log.L().Debug("trying to find image", "name", name)

	ref, err := reference.ParseAnyReference(name)
	if err != nil {
		return "", fmt.Errorf("could not parse image name: %w", err)
	}

	client, err := c.Client()
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

	configKey := repoInfo.Index.Name

	if repoInfo.Index.Official {
		info, err := client.Info(context.Background())
		if err != nil && info.IndexServerAddress != "" {
			configKey = info.IndexServerAddress
		} else {
			configKey = registry.IndexServer
		}
	}

	authConfigs := docker.LoadAuthConfig()

	buf, err := json.Marshal(authConfigs[configKey])
	if err != nil {
		return "", fmt.Errorf("could not parse auth config: %w", err)
	}

	response, err := client.ImagePull(
		context.Background(),
		parsed.String(),
		types.ImagePullOptions{
			RegistryAuth: base64.URLEncoding.EncodeToString(buf),
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
