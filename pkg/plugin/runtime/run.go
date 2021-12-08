package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-core/pkg/plugin/runtime"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/sync/errgroup"
)

func (c *ContainerRuntime) StartContainer(id string) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	return client.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (c *ContainerRuntime) RunAndWaitContainer(id string, height uint32, width uint32) (*runtime.Result, error) {
	client, err := c.ensureClient()
	if err != nil {
		return nil, err
	}

	waitCh, errorCh := client.ContainerWait(context.Background(), id, container.WaitConditionRemoved)

	if err := c.StartContainer(id); err != nil {
		return nil, fmt.Errorf("could not stop container: %w", err)
	}

	if height != 0 || width != 0 {
		if err := c.ResizeContainer(id, height, width); err != nil {
			log.L().Error("error during resize", "error", err)
		}
	}

	select {
	case resp := <-waitCh:
		if resp.Error != nil {
			return nil, errors.New(resp.Error.Message)
		}

		return &runtime.Result{ExitCode: int(resp.StatusCode)}, nil
	case err := <-errorCh:
		return nil, err
	}
}

func (c *ContainerRuntime) StreamContainer(id string, stream *plugin.StreamConfig) (*runtime.Result, error) {
	ctx := context.Background()

	s, err := c.AttachContainer(ctx, id, stream)
	if err != nil {
		return nil, err
	}

	eg, _ := errgroup.WithContext(ctx)
	result := &runtime.Result{}

	eg.Go(s.CopyOutput)
	eg.Go(s.CopyInput)

	eg.Go(func() error {
		r, err := c.RunAndWaitContainer(id, stream.TerminalHeight, stream.TerminalWidth)

		result.ExitCode = r.ExitCode

		return err
	})

	return result, eg.Wait()
}

func (c *ContainerRuntime) ResizeContainer(id string, height uint32, width uint32) error {
	client, err := c.ensureClient()
	if err != nil {
		return err
	}

	return client.ContainerResize(
		context.Background(),
		id,
		types.ResizeOptions{
			Height: uint(height),
			Width:  uint(width),
		},
	)
}
