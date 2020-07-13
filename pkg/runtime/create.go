package runtime

import (
	"fmt"
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-connections/nat"
	"github.com/oclaussen/dodo/pkg/types"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) CreateContainer(config *types.Backdrop, tty bool, stdio bool) (string, error) {
	// TODO: share tmpPath?
	tmpPath := fmt.Sprintf("/tmp/dodo-%s/", stringid.GenerateRandomID()[:20])
	entrypoint, command := entrypoint(config, tmpPath)

	response, err := c.client.ContainerCreate(
		context.Background(),
		&container.Config{
			User:         config.User,
			AttachStdin:  stdio,
			AttachStdout: stdio,
			AttachStderr: stdio,
			Tty:          tty && stdio,
			OpenStdin:    stdio,
			StdinOnce:    stdio,
			Env:          environment(config),
			Cmd:          command,
			Image:        config.ImageId,
			WorkingDir:   config.WorkingDir,
			Entrypoint:   entrypoint,
			ExposedPorts: portSet(config),
		},
		&container.HostConfig{
			AutoRemove: func() bool {
				return stdio
			}(),
			Binds:         volumes(config),
			PortBindings:  portMap(config),
			RestartPolicy: restartPolicy(stdio),
			Resources: container.Resources{
				Devices:           devices(config),
				DeviceCgroupRules: deviceCgroupRules(config),
			},
		},
		&network.NetworkingConfig{},
		config.ContainerName,
	)
	if err != nil {
		return "", err
	}

	if len(config.Entrypoint.Script) > 0 {
		if err := c.UploadFile(response.ID, path.Join(tmpPath, "entrypoint"), []byte(config.Entrypoint.Script+"\n")); err != nil {
			return "", err
		}
	}

	return response.ID, nil
}

func entrypoint(config *types.Backdrop, tmpPath string) ([]string, []string) {
	entrypoint := []string{"/bin/sh"}
	command := config.Entrypoint.Arguments

	if config.Entrypoint.Interpreter != nil {
		entrypoint = config.Entrypoint.Interpreter
	}

	if config.Entrypoint.Interactive {
		command = nil
	} else if len(config.Entrypoint.Script) > 0 {
		entrypoint = append(entrypoint, path.Join(tmpPath, "entrypoint"))
	}

	return entrypoint, command
}

func restartPolicy(stdio bool) container.RestartPolicy {
	if stdio {
		return container.RestartPolicy{Name: "no"}
	}

	return container.RestartPolicy{Name: "always"}
}

func devices(config *types.Backdrop) []container.DeviceMapping {
	result := []container.DeviceMapping{}

	for _, device := range config.Devices {
		if len(device.CgroupRule) > 0 {
			continue
		}

		result = append(result, container.DeviceMapping{
			PathOnHost:        device.Source,
			PathInContainer:   device.Target,
			CgroupPermissions: device.Permissions,
		})
	}

	return result
}

func deviceCgroupRules(config *types.Backdrop) []string {
	result := []string{}

	for _, device := range config.Devices {
		if len(device.CgroupRule) > 0 {
			result = append(result, device.CgroupRule)
		}
	}

	return result
}

func portMap(config *types.Backdrop) nat.PortMap {
	result := map[nat.Port][]nat.PortBinding{}

	for _, port := range config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.Target)
		result[portSpec] = append(result[portSpec], nat.PortBinding{HostPort: port.Published})
	}

	return result
}

func portSet(config *types.Backdrop) nat.PortSet {
	result := map[nat.Port]struct{}{}

	for _, port := range config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.Target)
		result[portSpec] = struct{}{}
	}

	return result
}

func environment(config *types.Backdrop) []string {
	result := []string{}

	for _, kv := range config.Environment {
		result = append(result, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
	}

	return result
}

func volumes(config *types.Backdrop) []string {
	result := []string{}

	for _, v := range config.Volumes {
		var volumeString string

		if v.Target == "" && !v.Readonly {
			volumeString = fmt.Sprintf("%s:%s", v.Source, v.Source)
		} else if !v.Readonly {
			volumeString = fmt.Sprintf("%s:%s", v.Source, v.Target)
		} else {
			volumeString = fmt.Sprintf("%s:%s:ro", v.Source, v.Target)
		}

		result = append(result, volumeString)
	}

	return result
}
