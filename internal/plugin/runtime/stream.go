package runtime

import (
	"context"
	"fmt"
	"io"

	log "github.com/hashicorp/go-hclog"
	"github.com/moby/moby/api/pkg/stdcopy"
	moby "github.com/moby/moby/client"
	"github.com/wabenet/dodo-core/pkg/ioutil"
	"github.com/wabenet/dodo-core/pkg/plugin"
)

type ContainerStream struct {
	hasTTY bool
	hijack moby.HijackedResponse

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type closeWriter interface {
	CloseWrite() error
}

func (c *ContainerRuntime) AttachContainer(
	ctx context.Context, id string, stream *plugin.StreamConfig,
) (*ContainerStream, func(), error) {
	client, err := c.ensureClient()
	if err != nil {
		return nil, nil, err
	}

	inspectResult, err := client.ContainerInspect(ctx, id, moby.ContainerInspectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("could not inspect container: %w", err)
	}

	attachResult, err := client.ContainerAttach(
		ctx,
		id,
		moby.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
			Logs:   true,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not attach to container: %w", err)
	}

	inContext, cancel := context.WithCancel(context.Background())
	inReader := ioutil.NewCancelableReader(inContext, stream.Stdin)

	return &ContainerStream{
		hasTTY: inspectResult.Container.Config.Tty,
		hijack: attachResult.HijackedResponse,
		stdin:  inReader,
		stdout: stream.Stdout,
		stderr: stream.Stderr,
	}, cancel, nil
}

func (s *ContainerStream) CopyOutput() error {
	if s.hasTTY {
		if _, err := io.Copy(s.stdout, s.hijack.Reader); err != nil {
			log.L().Warn("could not copy container output", "error", err)
		}
	} else {
		if _, err := stdcopy.StdCopy(s.stdout, s.stderr, s.hijack.Reader); err != nil {
			log.L().Warn("could not copy container output", "error", err)
		}
	}

	return nil
}

func (s *ContainerStream) CopyInput() error {
	defer func() {
		cw, ok := s.hijack.Conn.(closeWriter)
		if ok {
			if err := cw.CloseWrite(); err != nil {
				log.L().Warn("could not close streaming connection", "error", err)
			}
		} else {
			if err := s.hijack.Conn.Close(); err != nil {
				log.L().Warn("could not close streaming connection", "error", err)
			}
		}
	}()

	if _, err := io.Copy(s.hijack.Conn, s.stdin); err != nil {
		log.L().Error("could not copy container input", "error", err)
	}

	return nil
}
