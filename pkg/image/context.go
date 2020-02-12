package image

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/urlutil"
	buildkit "github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/session/sshforward/sshprovider"
	"github.com/oclaussen/dodo/pkg/types"
	"github.com/pkg/errors"
	fstypes "github.com/tonistiigi/fsutil/types"
)

const clientSession = "client-session"

type contextData struct {
	remote         string
	dockerfileName string
	contextDir     string
}

func (data *contextData) tempdir() (string, error) {
	if len(data.contextDir) == 0 {
		dir, err := ioutil.TempDir("", "dodo-temp-")
		if err != nil {
			return "", err
		}
		data.contextDir = dir
	}
	return data.contextDir, nil
}

func (data *contextData) cleanup() {
	if data.contextDir != "" {
		os.RemoveAll(data.contextDir)
	}
}

func prepareContext(config *types.BuildInfo, session session) (*contextData, error) {
	data := contextData{
		remote:         "",
		dockerfileName: config.Dockerfile,
	}
	syncedDirs := []filesync.SyncedDir{}

	if config.Context == "" {
		data.remote = clientSession
		dir, err := data.tempdir()
		if err != nil {
			data.cleanup()
			return nil, err
		}
		syncedDirs = append(syncedDirs, filesync.SyncedDir{Name: "context", Dir: dir})

	} else if _, err := os.Stat(config.Context); err == nil {
		data.remote = clientSession
		syncedDirs = append(syncedDirs, filesync.SyncedDir{
			Name: "context",
			Dir:  config.Context,
			Map: func(stat *fstypes.Stat) bool {
				stat.Uid = 0
				stat.Gid = 0
				return true
			},
		})

	} else if urlutil.IsURL(config.Context) {
		data.remote = config.Context

	} else {
		return nil, errors.Errorf("Context directory does not exist: %v", config.Context)
	}

	if len(config.InlineDockerfile) > 0 {
		steps := ""
		for _, step := range config.InlineDockerfile {
			steps = steps + step + "\n"
		}

		dir, err := data.tempdir()
		if err != nil {
			data.cleanup()
			return nil, err
		}
		tempfile := filepath.Join(dir, "Dockerfile")
		if err := writeDockerfile(tempfile, steps); err != nil {
			data.cleanup()
			return nil, err
		}

		data.dockerfileName = filepath.Base(tempfile)
		dockerfileDir := filepath.Dir(tempfile)
		syncedDirs = append(syncedDirs, filesync.SyncedDir{
			Name: "dockerfile",
			Dir:  dockerfileDir,
		})

	} else if config.Dockerfile != "" && data.remote == clientSession {
		data.dockerfileName = filepath.Base(config.Dockerfile)
		dockerfileDir := filepath.Dir(config.Dockerfile)
		syncedDirs = append(syncedDirs, filesync.SyncedDir{
			Name: "dockerfile",
			Dir:  dockerfileDir,
		})

	} else if config.ImageName != "" && data.remote == clientSession {
		dir, err := data.tempdir()
		if err != nil {
			data.cleanup()
			return nil, err
		}
		tempfile := filepath.Join(dir, "Dockerfile")
		if err := writeDockerfile(tempfile, fmt.Sprintf("FROM %s", config.ImageName)); err != nil {
			data.cleanup()
			return nil, err
		}
		data.dockerfileName = filepath.Base(tempfile)
		dockerfileDir := filepath.Dir(tempfile)
		syncedDirs = append(syncedDirs, filesync.SyncedDir{
			Name: "dockerfile",
			Dir:  dockerfileDir,
		})

	}

	if len(syncedDirs) > 0 {
		session.Allow(filesync.NewFSSyncProvider(syncedDirs))
	}

	session.Allow(authprovider.NewDockerAuthProvider())
	if len(config.Secrets) > 0 {
		provider, err := secretsProvider(config)
		if err != nil {
			return nil, err
		}
		session.Allow(provider)
	}
	if len(config.SshAgents) > 0 {
		provider, err := sshAgentProvider(config)
		if err != nil {
			return nil, err
		}
		session.Allow(provider)
	}

	return &data, nil
}

func writeDockerfile(path string, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	rc := ioutil.NopCloser(bytes.NewReader([]byte(content)))
	_, err = io.Copy(file, rc)
	if err != nil {
		return err
	}

	err = rc.Close()
	if err != nil {
		return err
	}

	return nil
}

func secretsProvider(config *types.BuildInfo) (buildkit.Attachable, error) {
	sources := make([]secretsprovider.FileSource, 0, len(config.Secrets))
	for _, secret := range config.Secrets {
		source := secretsprovider.FileSource{
			ID:       secret.Id,
			FilePath: secret.Path,
		}
		sources = append(sources, source)
	}
	store, err := secretsprovider.NewFileStore(sources)
	if err != nil {
		return nil, err
	}
	return secretsprovider.NewSecretProvider(store), nil
}

func sshAgentProvider(config *types.BuildInfo) (buildkit.Attachable, error) {
	configs := make([]sshprovider.AgentConfig, 0, len(config.SshAgents))
	for _, agent := range config.SshAgents {
		config := sshprovider.AgentConfig{
			ID:    agent.Id,
			Paths: []string{agent.IdentityFile},
		}
		configs = append(configs, config)
	}
	return sshprovider.NewSSHAgentProvider(configs)
}
