package runtime

import (
	"io"
	"net"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/dodo-cli/dodo-core/pkg/plugin/runtime"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) StartContainer(id string) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	return client.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (c *ContainerRuntime) StreamContainer(id string, r io.Reader, w io.Writer, height uint32, width uint32) error {
	ctx := context.Background()

	client, err := c.Client()
	if err != nil {
		return err
	}

	config, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return err
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
		return err
	}
	defer closeStreamingConnection(attach.Conn)

	outputDone := make(chan error)
	go func() {
		if config.Config.Tty {
			_, err := io.Copy(w, attach.Reader)
			outputDone <- err
		} else {
			// TODO: Write stderr to streaming connection.
			// Currently, this works if the plugin is compiled in,
			// but will fail over gcpr.
			_, err := stdcopy.StdCopy(w, os.Stderr, attach.Reader)
			outputDone <- err
		}
	}()

	inputDone := make(chan struct{})
	go func() {
		if _, err := io.Copy(attach.Conn, r); err != nil {
			log.L().Warn("could not copy container input", "error", err)
		}
		closeStreamingConnection(attach.Conn)
		close(inputDone)
	}()

	streamChan := make(chan error, 1)
	go func() {
		select {
		case err := <-outputDone:
			streamChan <- err
		case <-inputDone:
			select {
			case err := <-outputDone:
				streamChan <- err
			case <-ctx.Done():
				streamChan <- ctx.Err()
			}
		case <-ctx.Done():
			streamChan <- ctx.Err()
		}
	}()

	waitCh, errorCh := client.ContainerWait(ctx, id, container.WaitConditionRemoved)

	if err := c.StartContainer(id); err != nil {
		return err
	}

	if height != 0 || width != 0 {
		c.ResizeContainer(id, height, width)
	}

	if err := <-streamChan; err != nil {
		return err
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

func closeStreamingConnection(conn net.Conn) {
	log.L().Info("closing writer")
	if cw, ok := conn.(CloseWriter); ok {
		if err := cw.CloseWrite(); err != nil {
			log.L().Warn("could not close streaming connection", "error", err)
		}
	} else {
		if err := conn.Close(); err != nil {
			log.L().Warn("could not close streaming connection", "error", err)
		}
	}
}

// TODO there must be something easier
type CloseWriter interface {
	CloseWrite() error
}
