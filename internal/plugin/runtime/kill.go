package runtime

import (
	"os"

	"github.com/moby/sys/signal"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) KillContainer(id string, sig os.Signal) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	for str, sigN := range signal.SignalMap {
		if sigN == sig {
			return client.ContainerKill(context.Background(), id, str)
		}
	}

	return nil
}
