package container

import (
	"context"
	"fmt"
	"github.com/cecobask/spacelift-coding-challenge/pkg/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"strings"
)

type Docker struct {
	ctx    context.Context
	client *client.Client
	logger *log.Logger
}

func NewDocker(ctx context.Context, logger *log.Logger) (*Docker, error) {
	dockerClient, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}
	return &Docker{
		ctx:    ctx,
		client: dockerClient,
		logger: logger,
	}, nil
}

func (d *Docker) GetNetworkWithName(name string) (*types.NetworkResource, error) {
	networks, err := d.client.NetworkList(d.ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(
			filters.Arg("name", name),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("could not list networks: %w", err)
	}
	if len(networks) != 1 {
		return nil, fmt.Errorf("could not find network %s", name)
	}
	return &networks[0], nil
}

func (d *Docker) GetContainersWithLabel(label string) ([]types.ContainerJSON, error) {
	containerList, err := d.client.ContainerList(d.ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", label),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("could not list containers: %w", err)
	}
	if len(containerList) == 0 {
		return nil, fmt.Errorf("could not find containers with label %s", label)
	}
	var containers []types.ContainerJSON
	for i := range containerList {
		c, err := d.client.ContainerInspect(d.ctx, containerList[i].ID)
		if err != nil {
			return nil, fmt.Errorf("could not inspect container %s: %w", c.ID, err)
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func GetContainerEnvironmentVariables(config *container.Config) map[string]string {
	variables := make(map[string]string)
	for _, env := range config.Env {
		substrings := strings.Split(env, "=")
		if len(substrings) > 1 {
			variables[substrings[0]] = substrings[1]
		}
	}
	return variables
}
