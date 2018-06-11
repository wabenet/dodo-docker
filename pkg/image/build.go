package image

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"golang.org/x/net/context"
)

func build(ctx context.Context, options Options) (string, error) {
	args := map[string]*string{}
	for _, arg := range options.Args {
		switch values := strings.SplitN(arg, "=", 2); len(values) {
		case 1:
			args[values[0]] = nil
		case 2:
			args[values[0]] = &values[1]
		}
	}

	dockerfile := ".dockerfile." + stringid.GenerateRandomID()[:20]
	buildContext, err := getContext(options, dockerfile)
	if err != nil {
		return "", err
	}

	var tags []string
	if options.Name != "" {
		tags = append(tags, options.Name)
	}

	response, err := options.Client.ImageBuild(
		ctx,
		buildContext,
		types.ImageBuildOptions{
			Tags:           tags,
			SuppressOutput: false,
			NoCache:        options.NoCache,
			Remove:         true,
			ForceRemove:    true,
			PullParent:     options.ForcePull,
			Dockerfile:     dockerfile,
			BuildArgs:      args,
			AuthConfigs:    options.AuthConfigs,
		},
	)
	if err != nil {
		return "", err
	}

	name, err := handleImageResult(response.Body, true)
	if err != nil {
		return "", err
	}

	return name, nil
}