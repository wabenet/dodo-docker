package container

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/oclaussen/go-gimme/configfiles"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func (c *Container) GetImage() (string, error) {
	ref, err := reference.ParseAnyReference(c.config.ImageId)
	if err != nil {
		return "", err
	}

	if _, _, err := c.client.ImageInspectWithRaw(context.Background(), ref.String()); err == nil {
		return ref.String(), nil
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

	authConfigs := loadAuthConfig()
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

func loadAuthConfig() map[string]types.AuthConfig {
	var authConfigs map[string]types.AuthConfig

	configFile, err := configfiles.GimmeConfigFiles(&configfiles.Options{
		Name:       "docker",
		Extensions: []string{"json"},
		Filter: func(configFile *configfiles.ConfigFile) bool {
			var config map[string]*json.RawMessage
			err := json.Unmarshal(configFile.Content, &config)
			return err == nil && config["auths"] != nil
		},
	})
	if err != nil {
		return authConfigs
	}

	var config map[string]*json.RawMessage
	if err = json.Unmarshal(configFile.Content, &config); err != nil || config["auths"] == nil {
		return authConfigs
	}
	if err = json.Unmarshal(*config["auths"], &authConfigs); err != nil {
		return authConfigs
	}

	for addr, ac := range authConfigs {
		ac.Username, ac.Password, err = decodeAuth(ac.Auth)
		if err == nil {
			ac.Auth = ""
			ac.ServerAddress = addr
			authConfigs[addr] = ac
		}
	}

	return authConfigs
}

func decodeAuth(authStr string) (string, string, error) {
	if authStr == "" {
		return "", "", nil
	}

	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return "", "", err
	}
	if n > decLen {
		return "", "", errors.New("something went wrong decoding auth config")
	}
	arr := strings.SplitN(string(decoded), ":", 2)
	if len(arr) != 2 {
		return "", "", errors.New("invalid auth configuration file")
	}
	password := strings.Trim(arr[1], "\x00")
	return arr[0], password, nil
}
