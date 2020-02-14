package container

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"golang.org/x/net/context"
)

func (c *Container) GetImage() (string, error) {
	filter := filters.NewArgs(filters.Arg("reference", c.config.ImageId))
	images, err := c.client.ImageList(
		context.Background(),
		types.ImageListOptions{Filters: filter},
	)
	if err != nil {
		return "", err
	}
	if len(images) > 0 {
		return images[0].ID, nil
	}

	parsed, err := reference.ParseNormalizedNamed(c.config.ImageId)
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

	buf, err := json.Marshal(c.authConfigs[configKey])
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

func streamPull(result io.ReadCloser) error {
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
