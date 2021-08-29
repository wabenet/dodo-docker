package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	"github.com/dodo-cli/dodo-core/pkg/plugin/runtime"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/sync/errgroup"
)

func (c *ContainerRuntime) StartContainer(id string) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	return client.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (c *ContainerRuntime) RunAndWaitContainer(id string, height uint32, width uint32) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	waitCh, errorCh := client.ContainerWait(context.Background(), id, container.WaitConditionRemoved)

	if err := c.StartContainer(id); err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}

	if height != 0 || width != 0 {
		if err := c.ResizeContainer(id, height, width); err != nil {
			log.L().Error("error during resize", "error", err)
		}
	}

	select {
	case resp := <-waitCh:
		if resp.Error != nil {
			return &runtime.Result{
				Message:  resp.Error.Message,
				ExitCode: resp.StatusCode,
			}
		}

		return nil
	case err := <-errorCh:
		return err
	}
}

func (c *ContainerRuntime) StreamContainer(id string, stream *plugin.StreamConfig) error {
	ctx := context.Background()

	client, err := c.Client()
	if err != nil {
		return err
	}

	config, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return fmt.Errorf("could not inspect container: %w", err)
	}

	attach, err := client.ContainerAttach(
		ctx,
		id,
		types.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
			Logs:   true,
		},
	)
	if err != nil {
		return fmt.Errorf("could not attach to container: %w", err)
	}

	defer func() {
		if err := attach.Conn.Close(); err != nil {
			log.L().Warn("could not close streaming connection", "error", err)
		}
	}()

	eg, _ := errgroup.WithContext(ctx)
	inCopier := plugin.NewCancelCopier(stream.Stdin, attach.Conn)

	eg.Go(func() error {
		defer inCopier.Close()
		if config.Config.Tty {
			if _, err := io.Copy(stream.Stdout, attach.Reader); err != nil {
				log.L().Warn("could not copy container output", "error", err)
			}
		} else {
			if _, err := stdcopy.StdCopy(stream.Stdout, stream.Stderr, attach.Reader); err != nil {
				log.L().Warn("could not copy container output", "error", err)
			}
		}

		return nil
	})

	eg.Go(func() error {
		if err := inCopier.Copy(); err != nil {
			log.L().Error("could not copy container input", "error", err)
		}

		return nil
	})

	eg.Go(func() error {
		return c.RunAndWaitContainer(id, stream.TerminalHeight, stream.TerminalWidth)
	})

	return eg.Wait()
}

func (c *ContainerRuntime) ResizeContainer(id string, height uint32, width uint32) error {
	client, err := c.Client()
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
