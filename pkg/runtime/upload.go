package runtime

import (
	"archive/tar"
	"io"

	dockerapi "github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) UploadFile(containerID string, path string, contents []byte) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	go c.client.CopyToContainer(
		context.Background(),
		containerID,
		"/",
		reader,
		dockerapi.CopyToContainerOptions{},
	)

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

        err := tarWriter.WriteHeader(&tar.Header{
		Name: path,
		Mode: 0644,
		Size: int64(len(contents)),
	})
	if err != nil {
		return err
	}
	_, err = tarWriter.Write(contents)
	return err
}
