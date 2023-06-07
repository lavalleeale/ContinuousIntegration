package lib

import (
	"os"

	"github.com/docker/docker/client"
)

var Cli *client.Client

func StartClient() *client.Client {
	var err error
	host := os.Getenv("DOCKER_HOST")
	if host == "local" {
		Cli, err = client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	} else {
		Cli, err = client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithHost(host))
	}

	if err != nil {
		panic(err)
	}
	return Cli
}
