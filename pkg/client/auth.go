package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/oclaussen/go-gimme/configfiles"
)

type ConfigError string

const ErrInvalidAuthConfig ConfigError = "invalid auth config"

func (e ConfigError) Error() string {
	return string(e)
}

func LoadAuthConfig() map[string]types.AuthConfig {
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
	if err := json.Unmarshal(configFile.Content, &config); err != nil || config["auths"] == nil {
		return authConfigs
	}

	if err := json.Unmarshal(*config["auths"], &authConfigs); err != nil {
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
		return "", "", fmt.Errorf("invalid base64 string: %w", err)
	}

	if n > decLen {
		return "", "", ErrInvalidAuthConfig
	}

	arr := strings.SplitN(string(decoded), ":", 2)
	if len(arr) != 2 {
		return "", "", ErrInvalidAuthConfig
	}

	password := strings.Trim(arr[1], "\x00")

	return arr[0], password, nil
}
