package lib

import (
	"github.com/docker/docker/client"
)

var DockerCli *client.Client

func StartDockerClient() {
	var err error
	DockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		// Never expect docker to error
		panic(err)
	}
}
