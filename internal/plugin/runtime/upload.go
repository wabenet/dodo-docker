package runtime

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"golang.org/x/sync/errgroup"
)

const (
	filemode = 0644
)

func (c *ContainerRuntime) WriteFile(containerID string, path string, contents []byte) error {
	eg, ctx := errgroup.WithContext(context.Background())
	reader, writer := io.Pipe()

	eg.Go(func() error {
		defer reader.Close()

		client, err := c.ensureClient()
		if err != nil {
			return err
		}

		return client.CopyToContainer(
			ctx,
			containerID,
			"/",
			reader,
			types.CopyToContainerOptions{},
		)
	})

	eg.Go(func() error {
		tarWriter := tar.NewWriter(writer)
		defer tarWriter.Close()
		defer writer.Close()

		if err := tarWriter.WriteHeader(&tar.Header{
			Name: path,
			Mode: filemode,
			Size: int64(len(contents)),
		}); err != nil {
			return fmt.Errorf("could not write tar stream: %w", err)
		}

		if _, err := tarWriter.Write(contents); err != nil {
			return fmt.Errorf("could not write tar stream: %w", err)
		}

		return nil
	})

	return eg.Wait()
}
