package runtime

import (
	"context"
	"fmt"

	moby "github.com/moby/moby/client"
)

func (c *ContainerRuntime) ListVolumes() ([]string, error) {
	volumes := []string{}

	client, err := c.ensureClient()
	if err != nil {
		return volumes, err
	}

	resp, err := client.VolumeList(context.Background(), moby.VolumeListOptions{})
	if err != nil {
		return volumes, fmt.Errorf("could not list volumes: %w", err)
	}

	for _, item := range resp.Items {
		volumes = append(volumes, item.Name)
	}

	return volumes, nil
}

func (c *ContainerRuntime) CreateVolume(name string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	if _, err := client.VolumeCreate(
		context.Background(),
		moby.VolumeCreateOptions{
			Name: name,
		},
	); err != nil {
		return fmt.Errorf("could not create volume: %w", err)
	}

	return nil
}

func (c *ContainerRuntime) DeleteVolume(name string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	_, err = client.VolumeRemove(
		context.Background(),
		name,
		moby.VolumeRemoveOptions{
			Force: false,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete volume: %w", err)
	}

	return nil
}
