package lib

import (
	"bytes"
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
	"github.com/lavalleeale/ContinuousIntegration/db"
	"golang.org/x/exp/slices"
)

func StartBuild(repoUrl string, buildID uint, cont db.Container) {
	networkResp, err := Cli.NetworkCreate(context.Background(), fmt.Sprintf("%d", cont.Id), types.NetworkCreate{
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
					code := 255
					db.Db.Model(&cont).Updates(db.Container{Log: "Service container failed to be healthy", Code: &code})
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

	build := db.Build{ID: buildID}

	db.Db.Preload("Containers.UploadedFiles").First(&build)

	var mainEnv = make([]string, 0)
	if cont.Environment != nil {
		mainEnv = append(mainEnv, strings.Split(*cont.Environment, ",")...)
	}
	mainContainerResponse, err := Cli.ContainerCreate(context.TODO(), &container.Config{
		Image:  cont.Image,
		Cmd:    []string{"bash", "-c", fmt.Sprintf("GIT_SSL_NO_VERIFY=1 git clone %s %s /repo && cd /repo && %s", repoUrl, build.GitConfig, cont.Command)},
		Env:    mainEnv,
		Tty:    false,
		Labels: map[string]string{"buildId": fmt.Sprint(buildID), "containerId": fmt.Sprint(cont.Id)},
	}, &container.HostConfig{Runtime: os.Getenv("RUNTIME")}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkResp.ID: {
				Aliases: []string{cont.Name},
			},
		},
	}, nil, "")

	if err != nil {
		panic(err)
	}

	for _, neededFile := range cont.NeededFiles {
		neededCont := build.Containers[slices.IndexFunc(build.Containers, func(cont db.Container) bool { return cont.Name == neededFile.From })]
		// TODO: handle not found
		uploadedFile := neededCont.UploadedFiles[slices.IndexFunc(neededCont.UploadedFiles, func(file db.UploadedFile) bool { return file.Path == neededFile.FromPath })]
		err = Cli.CopyToContainer(context.TODO(), mainContainerResponse.ID, "/neededFiles/", bytes.NewReader(uploadedFile.Bytes), types.CopyToContainerOptions{})
		if err != nil {
			log.Println(err)
		}
	}

	if err := Cli.ContainerStart(context.TODO(), mainContainerResponse.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
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
		db.Db.Model(&cont).Updates(db.Container{Log: logString[0:24900] + "\nTruncated due to length over 25k chars", Code: &t.State.ExitCode})
	} else {
		db.Db.Model(&cont).Updates(db.Container{Log: logString, Code: &t.State.ExitCode})
	}
	for _, file := range cont.UploadedFiles {
		reader, _, err := Cli.CopyFromContainer(context.TODO(), mainContainerResponse.ID, file.Path)
		if err != nil {
			log.Println(err)
			db.Db.Model(&cont).Update("log", fmt.Sprintf("%s\nFailed to upload file (%s)", cont.Log, file.Path))
			return
		}
		defer reader.Close()
		bytes, err := io.ReadAll(reader)
		if err != nil {
			break
		}
		db.Db.Model(&file).Update("bytes", bytes)
	}
	Cli.ContainerRemove(context.TODO(), mainContainerResponse.ID, types.ContainerRemoveOptions{Force: true})
	if cont.ServiceImage != nil {
		Cli.ContainerRemove(context.TODO(), serviceContainerResponse.ID, types.ContainerRemoveOptions{Force: true})
	}
	Cli.NetworkRemove(context.TODO(), networkResp.ID)

	var edges []db.ContainerGraphEdge
	db.Db.Where(db.ContainerGraphEdge{FromID: uint(cont.Id)}, "FromID").Preload("To.EdgesToward.From").Preload("To.NeededFiles").Preload("To.UploadedFiles").Find(&edges)
	for _, edge := range edges {
		for index, from := range edge.To.EdgesToward {
			if from.From.Code == nil {
				break
			}
			if index == len(edge.To.EdgesToward)-1 {
				go StartBuild(repoUrl, buildID, edge.To)
			}
		}
	}
}
