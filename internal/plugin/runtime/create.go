package runtime

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/wabenet/dodo-core/pkg/plugin/runtime"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) CreateContainer(config runtime.ContainerConfig) (string, error) {
	client, err := c.ensureClient()
	if err != nil {
		return "", err
	}

	response, err := client.ContainerCreate(
		context.Background(),
		mkContainerConfig(config),
		mkHostConfig(config),
		mkNetworkingConfig(config),
		nil,
		config.Name,
	)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	return response.ID, nil
}

func mkContainerConfig(config runtime.ContainerConfig) *container.Config {
	return &container.Config{
		Image:        config.Image,
		AttachStdin:  config.Terminal.StdIO,
		AttachStdout: config.Terminal.StdIO,
		AttachStderr: config.Terminal.StdIO,
		OpenStdin:    config.Terminal.StdIO,
		StdinOnce:    config.Terminal.StdIO,
		Tty:          config.Terminal.TTY,
		User:         config.Process.User,
		WorkingDir:   config.Process.WorkingDir,
		Cmd:          []string(config.Process.Command),
		Entrypoint:   []string(config.Process.Entrypoint),
		Env:          mkEnvironment(config),
		ExposedPorts: mkPortSet(config),
	}
}

func mkHostConfig(config runtime.ContainerConfig) *container.HostConfig {
	return &container.HostConfig{
		AutoRemove: func() bool {
			return config.Terminal.StdIO
		}(),
		Mounts:        mkMounts(config),
		PortBindings:  mkPortMap(config),
		CapAdd:        []string(config.Capabilities),
		Init:          &[]bool{true}[0],
		RestartPolicy: mkRestartPolicy(config.Terminal.StdIO),
		Resources: container.Resources{
			Devices:           mkDevices(config),
			DeviceCgroupRules: mkDeviceCgroupRules(config),
		},
	}
}

func mkNetworkingConfig(_ runtime.ContainerConfig) *network.NetworkingConfig {
	return &network.NetworkingConfig{}
}

func mkRestartPolicy(stdio bool) container.RestartPolicy {
	if stdio {
		return container.RestartPolicy{Name: "no"}
	}

	return container.RestartPolicy{Name: "always"}
}

func mkPortMap(config runtime.ContainerConfig) nat.PortMap {
	result := map[nat.Port][]nat.PortBinding{}

	for _, port := range config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.ContainerPort)
		result[portSpec] = append(result[portSpec], nat.PortBinding{HostPort: port.HostPort})
	}

	return result
}

func mkPortSet(config runtime.ContainerConfig) nat.PortSet {
	result := map[nat.Port]struct{}{}

	for _, port := range config.Ports {
		portSpec, _ := nat.NewPort(port.Protocol, port.ContainerPort)
		result[portSpec] = struct{}{}
	}

	return result
}

func mkEnvironment(config runtime.ContainerConfig) []string {
	result := []string{}

	for _, kv := range config.Environment {
		result = append(result, kv.String())
	}

	return result
}

func mkMounts(config runtime.ContainerConfig) []mount.Mount {
	result := []mount.Mount{}

	for _, m := range config.Mounts {
		if r, ok := mkMount(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkMount(config runtime.Mount) (mount.Mount, bool) {
	switch config.Type() {
	case runtime.TypeBind:
		m, _ := config.(runtime.BindMount)
		return mkBindMount(m), true
	case runtime.TypeVolume:
		m, _ := config.(runtime.VolumeMount)
		return mkVolumeMount(m), true
	case runtime.TypeTmpfs:
		m, _ := config.(runtime.TmpfsMount)
		return mkTmpfsMount(m), true
	case runtime.TypeImage:
		m, _ := config.(runtime.ImageMount)
		return mkImageMount(m), true
	case runtime.TypeDevice:
		// Device mounts don't support the Mount API yet
		return mount.Mount{}, false
	default:
		return mount.Mount{}, false
	}
}

func mkDevices(config runtime.ContainerConfig) []container.DeviceMapping {
	result := []container.DeviceMapping{}

	for _, m := range config.Mounts {
		if r, ok := mkDevice(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkDevice(config runtime.Mount) (container.DeviceMapping, bool) {
	switch config.Type() {
	case runtime.TypeDevice:
		m, _ := config.(runtime.DeviceMount)
		return mkDeviceMount(m), true
	default:
		return container.DeviceMapping{}, false
	}
}

func mkDeviceCgroupRules(config runtime.ContainerConfig) []string {
	result := []string{}

	for _, m := range config.Mounts {
		if r, ok := mkDeviceCgroupRule(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkDeviceCgroupRule(config runtime.Mount) (string, bool) {
	switch config.Type() {
	case runtime.TypeDevice:
		m, _ := config.(runtime.DeviceMount)
		if len(m.CGroupRule) > 0 {
			return m.CGroupRule, true
		} else {
			return "", false
		}
	default:
		return "", false
	}
}

func mkBindMount(config runtime.BindMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeBind,
		Source:   config.HostPath,
		Target:   config.ContainerPath,
		ReadOnly: config.Readonly,
		BindOptions: &mount.BindOptions{
			CreateMountpoint: true,
		},
	}
}

func mkVolumeMount(config runtime.VolumeMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeVolume,
		Source:   config.VolumeName,
		Target:   config.ContainerPath,
		ReadOnly: config.Readonly,
		VolumeOptions: &mount.VolumeOptions{
			Subpath: config.Subpath,
		},
	}
}

func mkTmpfsMount(config runtime.TmpfsMount) mount.Mount {
	return mount.Mount{
		Type:   mount.TypeTmpfs,
		Target: config.ContainerPath,
		TmpfsOptions: &mount.TmpfsOptions{
			SizeBytes: int64(config.Size),
			Mode:      config.Mode,
		},
	}
}

func mkImageMount(config runtime.ImageMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeImage,
		Source:   config.Image,
		Target:   config.ContainerPath,
		ReadOnly: config.Readonly,
		ImageOptions: &mount.ImageOptions{
			Subpath: config.Subpath,
		},
	}
}

func mkDeviceMount(config runtime.DeviceMount) container.DeviceMapping {
	return container.DeviceMapping{
		PathOnHost:        config.HostPath,
		PathInContainer:   config.ContainerPath,
		CgroupPermissions: config.Permissions,
	}
}
