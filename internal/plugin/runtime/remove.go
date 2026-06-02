package runtime

import (
	"context"
	"fmt"

	moby "github.com/moby/moby/client"
)

func (c *ContainerRuntime) DeleteContainer(id string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	_, err = client.ContainerStop(context.Background(), id, moby.ContainerStopOptions{})
	if err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}

	_, err = client.ContainerRemove(context.Background(), id, moby.ContainerRemoveOptions{})
	if err != nil {
		return fmt.Errorf("could not remove container: %w", err)
	}

	return nil
}
