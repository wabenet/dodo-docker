package runtime

import (
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) CreateVolume(name string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	if _, err := client.VolumeCreate(
		context.Background(),
		volume.CreateOptions{
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

	if err := client.VolumeRemove(context.Background(), name, false); err != nil {
		return fmt.Errorf("could not delete volume: %w", err)
	}

	return nil
}
