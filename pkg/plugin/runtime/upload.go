package runtime

import (
	"archive/tar"
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"golang.org/x/sync/errgroup"
)

func (c *ContainerRuntime) UploadFile(containerID string, path string, contents []byte) error {
	eg, ctx := errgroup.WithContext(context.Background())
	reader, writer := io.Pipe()

	eg.Go(func() error {
		defer reader.Close()

		client, err := c.Client()
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
			Mode: 0644,
			Size: int64(len(contents)),
		}); err != nil {
			return err
		}

		_, err := tarWriter.Write(contents)

		return err
	})

	return eg.Wait()
}
