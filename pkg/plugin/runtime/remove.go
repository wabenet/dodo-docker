package runtime

import (
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) DeleteContainer(id string) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	if err := client.ContainerStop(context.Background(), id, nil); err != nil {
		return err
	}

	if err := client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	return nil
}
