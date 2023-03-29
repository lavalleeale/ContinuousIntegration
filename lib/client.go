package lib

import (
	"os"

	"github.com/docker/docker/client"
)

var Cli *client.Client

func StartClient() *client.Client {
	var err error
	Cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(os.Getenv("DOCKER_HOST")))

	if err != nil {
		panic(err)
	}
	return Cli
}
