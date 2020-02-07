package container

import (
	"archive/tar"
	"io"
	"path"

	dockerapi "github.com/docker/docker/api/types"
)

func (container *Container) UploadFile(containerID string, name string, contents []byte) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	go container.client.CopyToContainer(
		container.context,
		containerID,
		"/",
		reader,
		dockerapi.CopyToContainerOptions{},
	)

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	err := tarWriter.WriteHeader(&tar.Header{
		Name: path.Join(container.tmpPath, name),
		Mode: 0644,
		Size: int64(len(contents)),
	})
	if err != nil {
		return err
	}
	_, err = tarWriter.Write(contents)
	return err
}
