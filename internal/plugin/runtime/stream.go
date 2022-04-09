package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/dodo-cli/dodo-core/pkg/plugin"
	log "github.com/hashicorp/go-hclog"
)

type ContainerStream struct {
	hasTTY   bool
	config   *plugin.StreamConfig
	hijack   types.HijackedResponse
	inCopier *plugin.CancelCopier
}

type closeWriter interface {
	CloseWrite() error
}

func (c *ContainerRuntime) AttachContainer(
	ctx context.Context, id string, stream *plugin.StreamConfig,
) (*ContainerStream, error) {
	client, err := c.ensureClient()
	if err != nil {
		return nil, err
	}

	config, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not inspect container: %w", err)
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
		return nil, fmt.Errorf("could not attach to container: %w", err)
	}

	return &ContainerStream{
		hasTTY:   config.Config.Tty,
		config:   stream,
		hijack:   attach,
		inCopier: plugin.NewCancelCopier(stream.Stdin, attach.Conn),
	}, nil
}

func (s *ContainerStream) CopyOutput() error {
	defer s.inCopier.Close()

	if s.hasTTY {
		if _, err := io.Copy(s.config.Stdout, s.hijack.Reader); err != nil {
			log.L().Warn("could not copy container output", "error", err)
		}
	} else {
		if _, err := stdcopy.StdCopy(s.config.Stdout, s.config.Stderr, s.hijack.Reader); err != nil {
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

	if err := s.inCopier.Copy(); err != nil {
		log.L().Error("could not copy container input", "error", err)
	}

	return nil
}
