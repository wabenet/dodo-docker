package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/dodo/dodo-docker/pkg/client"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) ResolveImage(name string) (string, error) {
	log.L().Debug("trying to find image", "name", name)

	ref, err := reference.ParseAnyReference(name)
	if err != nil {
		return "", err
	}

	if _, _, err := c.client.ImageInspectWithRaw(context.Background(), ref.String()); err == nil {
		log.L().Debug("found image locally", "ref", ref.String())
		return ref.String(), nil
	}

	parsed, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return "", err
	}

	if reference.IsNameOnly(parsed) {
		parsed = reference.TagNameOnly(parsed)
	}

	repoInfo, err := registry.ParseRepositoryInfo(parsed)
	if err != nil {
		return "", err
	}

	configKey := repoInfo.Index.Name

	if repoInfo.Index.Official {
		info, err := c.client.Info(context.Background())
		if err != nil && info.IndexServerAddress != "" {
			configKey = info.IndexServerAddress
		} else {
			configKey = registry.IndexServer
		}
	}

	authConfigs := client.LoadAuthConfig()

	buf, err := json.Marshal(authConfigs[configKey])
	if err != nil {
		return "", err
	}

	response, err := c.client.ImagePull(
		context.Background(),
		parsed.String(),
		types.ImagePullOptions{
			RegistryAuth: base64.URLEncoding.EncodeToString(buf),
		},
	)
	if err != nil {
		return "", err
	}
	defer response.Close()

	if err = streamPull(response); err != nil {
		return "", err
	}

	return parsed.String(), nil
}

func streamPull(result io.Reader) error {
	decoder := json.NewDecoder(result)

	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		if msg.Error != nil {
			return msg.Error
		}

		if msg.Progress != nil || msg.ProgressMessage != "" {
			continue
		}

		if msg.Stream != "" {
			fmt.Fprintf(os.Stderr, "%s\n", msg.Stream)
		} else if msg.Status != "" {
			fmt.Fprintf(os.Stderr, "%s\n", msg.Status)
		}
	}

	return nil
}
