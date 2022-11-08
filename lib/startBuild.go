package lib

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
)

func StartBuild(repoUrl string, buildID int, cont db.Container) {

	// TODO: Make bridge net
	networkResp, err := Cli.NetworkCreate(context.Background(), "bridge", types.NetworkCreate{})
	if err != nil {
		log.Fatal(err)
	}

	containerResp, err := Cli.ContainerCreate(context.TODO(), &container.Config{
		Image:    "node:16",
		Cmd:      []string{"bash", "-c", fmt.Sprintf("git clone %s repo && cd repo && %s", repoUrl, cont.Command)},
		Tty:      false,
		Labels:   map[string]string{"buildId": fmt.Sprint(buildID), "containerId": fmt.Sprint(cont.ID)},
		Hostname: "build",
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"bridge": {
				NetworkID: networkResp.ID,
			},
		},
	}, nil, "")

	if err != nil {
		panic(err)
	}

	if err := Cli.ContainerStart(context.TODO(), containerResp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	go func() {
		logString := ""
		statusCh, errCh := Cli.ContainerWait(context.TODO(), containerResp.ID, container.WaitConditionNextExit)
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:
		}

		logs, err := Cli.ContainerLogs(context.TODO(), containerResp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			panic(err)
		}
		for {
			p := make([]byte, 1)

			_, err = logs.Read(p)
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}

			_, err = logs.Read(make([]byte, 3))
			if err != nil {
				panic(err)
			}

			var length int32
			binary.Read(logs, binary.BigEndian, &length)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println(err)
				os.Exit(1)
			}
			p = make([]byte, length)
			n, _ := logs.Read(p)
			logString += string(p[:n])
		}
		t, err := Cli.ContainerInspect(context.TODO(), containerResp.ID)
		if err != nil {
			panic(err)
		}

		if len(logString) > 25000 {
			db.Db.Update(context.TODO(), &cont, rel.Set("log", logString[0:24950]+"\nTruncated due to length over 25k chars"), rel.Set("code", t.State.ExitCode))
		} else {
			db.Db.Update(context.TODO(), &cont, rel.Set("log", logString), rel.Set("code", t.State.ExitCode))
		}
		Cli.ContainerRemove(context.TODO(), containerResp.ID, types.ContainerRemoveOptions{Force: true})
		Cli.NetworkRemove(context.TODO(), networkResp.ID)
	}()
}
