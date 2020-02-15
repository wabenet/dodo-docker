package container

import (
	"fmt"
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/oclaussen/dodo/pkg/plugin"
)

func (c *Container) create(image string) (string, error) {
	entrypoint, command := c.dockerEntrypoint()
	response, err := c.client.ContainerCreate(
		c.context,
		&container.Config{
			User:         c.config.User,
			AttachStdin:  !c.daemon,
			AttachStdout: !c.daemon,
			AttachStderr: !c.daemon,
			Tty:          hasTTY() && !c.daemon,
			OpenStdin:    !c.daemon,
			StdinOnce:    !c.daemon,
			Env:          c.dockerEnvironment(),
			Cmd:          command,
			Image:        image,
			WorkingDir:   c.config.WorkingDir,
			Entrypoint:   entrypoint,
			ExposedPorts: c.dockerPortSet(),
		},
		&container.HostConfig{
			AutoRemove: func() bool {
				return !c.daemon
			}(),
			Binds:         c.dockerVolumes(),
			PortBindings:  c.dockerPortMap(),
			RestartPolicy: c.dockerRestartPolicy(),
			Resources: container.Resources{
				Devices:           c.dockerDevices(),
				DeviceCgroupRules: c.dockerDeviceCgroupRules(),
			},
		},
		&network.NetworkingConfig{},
		c.name,
	)
	if err != nil {
		return "", err
	}

	if len(c.config.Entrypoint.Script) > 0 {
		if err := c.UploadFile(response.ID, "entrypoint", []byte(c.config.Entrypoint.Script+"\n")); err != nil {
			return "", err
		}
	}

	for _, pluginConfig := range plugin.GetConfigurations() {
		if err := pluginConfig.Provision(response.ID); err != nil {
			return "", err
		}
	}

	return response.ID, nil
}

func (c *Container) dockerEntrypoint() ([]string, []string) {
	entrypoint := []string{"/bin/sh"}
	command := c.config.Entrypoint.Arguments

	if c.config.Entrypoint.Interpreter != nil {
		entrypoint = c.config.Entrypoint.Interpreter
	}
	if c.config.Entrypoint.Interactive {
		command = nil
	} else if len(c.config.Entrypoint.Script) > 0 {
		entrypoint = append(entrypoint, path.Join(c.tmpPath, "entrypoint"))
	}

	return entrypoint, command
}

func (c *Container) dockerRestartPolicy() container.RestartPolicy {
	if c.daemon {
		return container.RestartPolicy{Name: "always"}
	} else {
		return container.RestartPolicy{Name: "no"}
	}
}

func (c *Container) dockerDevices() []container.DeviceMapping {
	result := []container.DeviceMapping{}
	for _, device := range c.config.Devices {
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

func (c *Container) dockerDeviceCgroupRules() []string {
	result := []string{}
	for _, device := range c.config.Devices {
		if len(device.CgroupRule) > 0 {
			result = append(result, device.CgroupRule)
		}
	}
	return result
}

func (c *Container) dockerPortMap() nat.PortMap {
	result := map[nat.Port][]nat.PortBinding{}
	for _, port := range c.config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.Target)
		result[portSpec] = append(result[portSpec], nat.PortBinding{HostPort: port.Published})
	}
	return result
}

func (c *Container) dockerPortSet() nat.PortSet {
	result := map[nat.Port]struct{}{}
	for _, port := range c.config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.Target)
		result[portSpec] = struct{}{}
	}
	return result
}

func (c *Container) dockerEnvironment() []string {
	result := []string{}
	for _, kv := range c.config.Environment {
		result = append(result, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
	}
	return result
}

func (c *Container) dockerVolumes() []string {
	result := []string{}
	for _, v := range c.config.Volumes {
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
