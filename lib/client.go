package lib

import (
	"github.com/docker/docker/client"
)

var Cli *client.Client

func StartClient() *client.Client {
	var err error
	Cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		panic(err)
	}
	return Cli
}
