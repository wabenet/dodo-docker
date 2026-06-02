package runtime

import (
	"context"
	"fmt"
	"os"

	moby "github.com/moby/moby/client"
	"github.com/moby/sys/signal"
)

func (c *ContainerRuntime) KillContainer(id string, sig os.Signal) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	for str, sigN := range signal.SignalMap {
		if sigN == sig {
			_, err := client.ContainerKill(
				context.Background(),
				id,
				moby.ContainerKillOptions{Signal: str},
			)

			return fmt.Errorf("could not kill container: %w", err)
		}
	}

	return nil
}
