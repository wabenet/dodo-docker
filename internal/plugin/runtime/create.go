package runtime

import (
	"fmt"
	"os"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	api "github.com/wabenet/dodo-core/api/runtime/v1alpha2"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) CreateContainer(config *api.ContainerConfig) (string, error) {
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
		config.GetName(),
	)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	return response.ID, nil
}

func mkContainerConfig(config *api.ContainerConfig) *container.Config {
	return &container.Config{
		Image:        config.GetImage(),
		AttachStdin:  config.GetTerminal().GetStdio(),
		AttachStdout: config.GetTerminal().GetStdio(),
		AttachStderr: config.GetTerminal().GetStdio(),
		OpenStdin:    config.GetTerminal().GetStdio(),
		StdinOnce:    config.GetTerminal().GetStdio(),
		Tty:          config.GetTerminal().GetTty(),
		User:         config.GetProcess().GetUser(),
		WorkingDir:   config.GetProcess().GetWorkingDir(),
		Cmd:          config.GetProcess().GetCommand(),
		Entrypoint:   config.GetProcess().GetEntrypoint(),
		Env:          mkEnvironment(config),
		ExposedPorts: mkPortSet(config),
	}
}

func mkHostConfig(config *api.ContainerConfig) *container.HostConfig {
	return &container.HostConfig{
		AutoRemove: func() bool {
			return config.GetTerminal().GetStdio()
		}(),
		Mounts:        mkMounts(config),
		PortBindings:  mkPortMap(config),
		CapAdd:        config.GetCapabilities(),
		Init:          &[]bool{true}[0],
		RestartPolicy: mkRestartPolicy(config.GetTerminal().GetStdio()),
		Resources: container.Resources{
			Devices:           mkDevices(config),
			DeviceCgroupRules: mkDeviceCgroupRules(config),
		},
	}
}

func mkNetworkingConfig(_ *api.ContainerConfig) *network.NetworkingConfig {
	return &network.NetworkingConfig{}
}

func mkRestartPolicy(stdio bool) container.RestartPolicy {
	if stdio {
		return container.RestartPolicy{Name: "no"}
	}

	return container.RestartPolicy{Name: "always"}
}

func mkPortMap(config *api.ContainerConfig) nat.PortMap {
	result := map[nat.Port][]nat.PortBinding{}

	for _, port := range config.GetPorts() {
		portSpec, _ := nat.NewPort(port.GetProtocol(), port.GetContainerPort())
		result[portSpec] = append(result[portSpec], nat.PortBinding{HostPort: port.GetHostPort()})
	}

	return result
}

func mkPortSet(config *api.ContainerConfig) nat.PortSet {
	result := map[nat.Port]struct{}{}

	for _, port := range config.GetPorts() {
		portSpec, _ := nat.NewPort(port.GetProtocol(), port.GetContainerPort())
		result[portSpec] = struct{}{}
	}

	return result
}

func mkEnvironment(config *api.ContainerConfig) []string {
	result := []string{}

	for _, kv := range config.GetEnvironment() {
		result = append(result, fmt.Sprintf("%s=%s", kv.GetKey(), kv.GetValue()))
	}

	return result
}

func mkMounts(config *api.ContainerConfig) []mount.Mount {
	result := []mount.Mount{}

	for _, m := range config.GetMounts() {
		if r, ok := mkMount(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkMount(config *api.Mount) (mount.Mount, bool) {
	switch m := config.GetType().(type) {
	case *api.Mount_Bind:
		return mkBindMount(m.Bind), true
	case *api.Mount_Volume:
		return mkVolumeMount(m.Volume), true
	case *api.Mount_Tmpfs:
		return mkTmpfsMount(m.Tmpfs), true
	case *api.Mount_Image:
		return mkImageMount(m.Image), true
	case *api.Mount_Device:
		// Device mounts don't support the Mount API yet
		return mount.Mount{}, false
	default:
		return mount.Mount{}, false
	}
}

func mkDevices(config *api.ContainerConfig) []container.DeviceMapping {
	result := []container.DeviceMapping{}

	for _, m := range config.GetMounts() {
		if r, ok := mkDevice(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkDevice(config *api.Mount) (container.DeviceMapping, bool) {
	switch m := config.GetType().(type) {
	case *api.Mount_Device:
		return mkDeviceMount(m.Device), true
	default:
		return container.DeviceMapping{}, false
	}
}

func mkDeviceCgroupRules(config *api.ContainerConfig) []string {
	result := []string{}

	for _, m := range config.GetMounts() {
		if r, ok := mkDeviceCgroupRule(m); ok {
			result = append(result, r)
		}
	}

	return result
}

func mkDeviceCgroupRule(config *api.Mount) (string, bool) {
	switch m := config.GetType().(type) {
	case *api.Mount_Device:
		if len(m.Device.GetCgroupRule()) > 0 {
			return m.Device.GetCgroupRule(), true
		} else {
			return "", false
		}
	default:
		return "", false
	}
}

func mkBindMount(config *api.BindMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeBind,
		Source:   config.GetHostPath(),
		Target:   config.GetContainerPath(),
		ReadOnly: config.GetReadonly(),
		BindOptions: &mount.BindOptions{
			CreateMountpoint: true,
		},
	}
}

func mkVolumeMount(config *api.VolumeMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeVolume,
		Source:   config.GetVolumeName(),
		Target:   config.GetContainerPath(),
		ReadOnly: config.GetReadonly(),
		VolumeOptions: &mount.VolumeOptions{
			Subpath: config.GetSubpath(),
		},
	}
}

func mkTmpfsMount(config *api.TmpfsMount) mount.Mount {
	mode, _ := strconv.ParseUint(config.GetMode(), 8, 32)
	// TODO error handling

	return mount.Mount{
		Type:   mount.TypeTmpfs,
		Target: config.GetContainerPath(),
		TmpfsOptions: &mount.TmpfsOptions{
			SizeBytes: config.GetSize(),
			Mode:      os.FileMode(mode),
		},
	}
}

func mkImageMount(config *api.ImageMount) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeImage,
		Source:   config.GetImage(),
		Target:   config.GetContainerPath(),
		ReadOnly: config.GetReadonly(),
		ImageOptions: &mount.ImageOptions{
			Subpath: config.GetSubpath(),
		},
	}
}

func mkDeviceMount(config *api.DeviceMount) container.DeviceMapping {
	return container.DeviceMapping{
		PathOnHost:        config.GetHostPath(),
		PathInContainer:   config.GetContainerPath(),
		CgroupPermissions: config.GetPermissions(),
	}
}
