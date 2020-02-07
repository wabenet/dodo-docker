package container

import (
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
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
			Env:          c.config.Environment.Strings(),
			Cmd:          command,
			Image:        image,
			WorkingDir:   c.config.WorkingDir,
			Entrypoint:   entrypoint,
			ExposedPorts: c.config.Ports.PortSet(),
		},
		&container.HostConfig{
			AutoRemove: func() bool {
				if c.daemon {
					return false
				}
				if c.config.Remove == nil {
					return true
				}
				return *c.config.Remove
			}(),
			Binds:         c.config.Volumes.Strings(),
			VolumesFrom:   c.config.VolumesFrom,
			PortBindings:  c.config.Ports.PortMap(),
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

	if len(c.config.Script) > 0 {
		if err := c.UploadFile(response.ID, "entrypoint", []byte(c.config.Script+"\n")); err != nil {
			return "", err
		}
	}

	return response.ID, nil
}

func (c *Container) dockerEntrypoint() ([]string, []string) {
	entrypoint := []string{"/bin/sh"}
	command := c.config.Command

	if c.config.Interpreter != nil {
		entrypoint = c.config.Interpreter
	}
	if c.config.Interactive {
		command = nil
	} else if len(c.config.Script) > 0 {
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
