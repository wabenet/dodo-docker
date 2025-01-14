package runtime

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) DeleteContainer(id string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	if err := client.ContainerStop(context.Background(), id, container.StopOptions{}); err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}

	return client.ContainerRemove(context.Background(), id, container.RemoveOptions{})
}
