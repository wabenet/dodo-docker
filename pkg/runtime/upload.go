package runtime

import (
	"archive/tar"
	"io"

	"github.com/docker/docker/api/types"
	log "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
)

func (c *ContainerRuntime) UploadFile(containerID string, path string, contents []byte) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	client, err := c.Client()
	if err != nil {
		return err
	}

	go func() {
		if err := client.CopyToContainer(
			context.Background(),
			containerID,
			"/",
			reader,
			types.CopyToContainerOptions{},
		); err != nil {
			log.L().Error("could not upload file to container", "error", err)
		}
	}()

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	if err := tarWriter.WriteHeader(&tar.Header{
		Name: path,
		Mode: 0644,
		Size: int64(len(contents)),
	}); err != nil {
		return err
	}

	_, err = tarWriter.Write(contents)

	return err
}
