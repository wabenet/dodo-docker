package runtime

import (
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	dodo "github.com/oclaussen/dodo/pkg/types"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) StartContainer(id string) error {
	return c.client.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (c *ContainerRuntime) StreamContainer(id string, r io.Reader, w io.Writer) error {
	ctx := context.Background()

	config, err := c.client.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	attach, err := c.client.ContainerAttach(
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

	if cw, ok := attach.Conn.(CloseWriter); ok {
		defer cw.CloseWrite()
	} else {
		defer attach.Conn.Close()
	}

	if config.Config.Tty {
		go io.Copy(w, attach.Reader)
	} else {
		// TODO: stderr
		go stdcopy.StdCopy(w, ioutil.Discard, attach.Reader)
	}

	go io.Copy(attach.Conn, r)

	waitCh, errorCh := c.client.ContainerWait(ctx, id, container.WaitConditionRemoved)

	if err := c.StartContainer(id); err != nil {
		return err
	}

	select {
	case resp := <-waitCh:
		if resp.Error != nil {
			return &dodo.Result{
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
	return c.client.ContainerResize(
		context.Background(),
		id,
		types.ResizeOptions{
			Height: uint(height),
			Width:  uint(width),
		},
	)
}

// TODO there must be something easier
type CloseWriter interface {
	CloseWrite() error
}
