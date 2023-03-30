package lib

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
)

func StartBuild(repoUrl string, buildID int, cont db.Container) {

	networkResp, err := Cli.NetworkCreate(context.Background(), fmt.Sprintf("%d", cont.ID), types.NetworkCreate{
		Driver: "bridge",
	})
	if err != nil {
		log.Fatal(err)
	}

	var serviceContainerResponse container.ContainerCreateCreatedBody

	if cont.ServiceImage != nil {
		// Create a container
		var givenCmd strslice.StrSlice
		if cont.ServiceCommand == nil {
			givenCmd = nil
		} else {
			givenCmd = []string{"bash", "-c", *cont.ServiceCommand}
		}
		var serivceEnv = make([]string, 0)
		if cont.ServiceEnvironment != nil {
			serivceEnv = append(serivceEnv, strings.Split(*cont.ServiceEnvironment, ",")...)
		}
		serviceContainerResponse, err = Cli.ContainerCreate(context.Background(), &container.Config{
			Image:  *cont.ServiceImage,
			Cmd:    givenCmd,
			Env:    serivceEnv,
			Labels: map[string]string{"buildId": fmt.Sprint(buildID)},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", *cont.ServiceHealthcheck},
				StartPeriod: time.Second * 30,
				Timeout:     time.Second * 15,
				Interval:    time.Second * 5,
				Retries:     5,
			},
		}, nil, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkResp.ID: {
					Aliases: []string{"service"},
				},
			},
		}, nil, "")

		if err != nil {
			panic(err)
		}

		if err := Cli.ContainerStart(context.TODO(), serviceContainerResponse.ID, types.ContainerStartOptions{}); err != nil {
			panic(err)
		}

		ctx, close := context.WithTimeout(context.TODO(), time.Minute)

		msgs, errs := Cli.Events(ctx, types.EventsOptions{
			Filters: filters.NewArgs(filters.Arg("container", serviceContainerResponse.ID), filters.Arg("event", "health_status")),
		})

	outer:
		for {
			select {
			case err := <-errs:
				log.Println(err)
				close()
				if errors.Is(err, context.DeadlineExceeded) {
					db.Db.Update(context.TODO(), &cont, rel.Set("log", "Service container failed to be healthy"), rel.Set("code", 255))
					return
				}
			case msg := <-msgs:
				if msg.Action == "health_status: healthy" {
					close()
					break outer
				}
			}
		}
	}

	var mainEnv = make([]string, 0)
	if cont.Environment != nil {
		mainEnv = append(mainEnv, strings.Split(*cont.Environment, ",")...)
	}
	mainContainerResponse, err := Cli.ContainerCreate(context.TODO(), &container.Config{
		Image:  cont.Image,
		Cmd:    []string{"bash", "-c", fmt.Sprintf("GIT_SSL_NO_VERIFY=1 git clone %s repo && cd repo && %s", repoUrl, cont.Command)},
		Env:    mainEnv,
		Tty:    false,
		Labels: map[string]string{"buildId": fmt.Sprint(buildID), "containerId": fmt.Sprint(cont.ID)},
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkResp.ID: {
				Aliases: []string{cont.Name},
			},
		},
	}, nil, "")

	if err != nil {
		panic(err)
	}

	if err := Cli.ContainerStart(context.TODO(), mainContainerResponse.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	go func() {
		logString := ""
		statusCh, errCh := Cli.ContainerWait(context.TODO(), mainContainerResponse.ID, container.WaitConditionNextExit)
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:
		}

		logs, err := Cli.ContainerLogs(context.TODO(), mainContainerResponse.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
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
		t, err := Cli.ContainerInspect(context.TODO(), mainContainerResponse.ID)
		if err != nil {
			panic(err)
		}

		if len(logString) > 25000 {
			db.Db.Update(context.TODO(), &cont, rel.Set("log", logString[0:24950]+"\nTruncated due to length over 25k chars"), rel.Set("code", t.State.ExitCode))
		} else {
			db.Db.Update(context.TODO(), &cont, rel.Set("log", logString), rel.Set("code", t.State.ExitCode))
		}
		Cli.ContainerRemove(context.TODO(), mainContainerResponse.ID, types.ContainerRemoveOptions{Force: true})
		if cont.ServiceImage != nil {
			Cli.ContainerRemove(context.TODO(), serviceContainerResponse.ID, types.ContainerRemoveOptions{Force: true})
		}
		Cli.NetworkRemove(context.TODO(), networkResp.ID)
	}()
}
