package lib

import (
	"os"

	"github.com/docker/docker/client"
)

var DockerCli *client.Client

func StartDockerClient() {
	var err error
	host := os.Getenv("DOCKER_HOST")
	if host == "local" {
		DockerCli, err = client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	} else {
		DockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	}

	if err != nil {
		panic(err)
	}
}
