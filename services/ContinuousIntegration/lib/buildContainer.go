package lib

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	sessionseal "github.com/lavalleeale/SessionSeal"
	"golang.org/x/exp/slices"
)

func BuildContainer(repoUrl string, buildID uint, cont db.Container, organizationId string, wg *sync.WaitGroup, failed *bool) {
	defer wg.Done()
	networkResp, err := DockerCli.NetworkCreate(context.Background(),
		fmt.Sprintf("%d", cont.Id), types.NetworkCreate{
			Driver: "bridge",
		})
	if err != nil {
		log.Fatal(err)
	}

	var serviceContainerResponse container.CreateResponse

	if cont.ServiceImage != nil {
		// Create a container
		var givenCmd strslice.StrSlice
		if cont.ServiceCommand == nil {
			givenCmd = nil
		} else {
			givenCmd = []string{"bash", "-c", *cont.ServiceCommand}
		}
		serivceEnv := make([]string, 0)
		if cont.ServiceEnvironment != nil {
			serivceEnv = append(serivceEnv, strings.Split(*cont.ServiceEnvironment, ",")...)
		}
		serviceContainerResponse, err = DockerCli.ContainerCreate(context.Background(), &container.Config{
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
			// Never expect docker to error
			panic(err)
		}

		if err := DockerCli.ContainerStart(context.TODO(), serviceContainerResponse.ID,
			types.ContainerStartOptions{}); err != nil {
			// Never expect docker to error
			panic(err)
		}

		ctx, close := context.WithTimeout(context.TODO(), time.Minute)

		msgs, errs := DockerCli.Events(ctx, types.EventsOptions{
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
					db.Db.Model(&cont).Updates(db.Container{
						Log: "Service container failed to be healthy", Code: &code,
					})
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

	var dockerAuth struct {
		OrganizationID string `json:"organizationId"`
	}

	dockerAuth.OrganizationID = organizationId

	dockerAuthData, _ := json.Marshal(dockerAuth)

	mainEnv := []string{
		"DOCKER_USER=token", fmt.Sprintf("DOCKER_PASS=%s",
			sessionseal.Seal(os.Getenv("JWT_SECRET"), dockerAuthData)),
		fmt.Sprintf("REGISTRY=%s", os.Getenv("REGISTRY_URL")),
	}
	if cont.Environment != nil {
		mainEnv = append(mainEnv, strings.Split(*cont.Environment, ",")...)
	}
	mainContainerResponse, err := DockerCli.ContainerCreate(context.TODO(), &container.Config{
		Image: cont.Image,
		Cmd: []string{"bash", "-c", fmt.Sprintf("GIT_SSL_NO_VERIFY=1 git clone %s %s /repo && cd /repo && %s",
			repoUrl, build.GitConfig, cont.Command)},
		Env:    mainEnv,
		Tty:    false,
		Labels: map[string]string{"buildId": fmt.Sprint(buildID), "containerId": fmt.Sprint(cont.Id)},
	}, &container.HostConfig{
		Runtime: os.Getenv("RUNTIME"),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkResp.ID: {
				Aliases: []string{cont.Name},
			},
		},
	}, nil, "")
	if err != nil {
		// Never expect docker to error
		panic(err)
	}

	for _, neededFile := range cont.NeededFiles {
		neededCont := build.Containers[slices.IndexFunc(build.Containers,
			func(cont db.Container) bool { return cont.Name == neededFile.From })]
		uploadedFile := neededCont.UploadedFiles[slices.IndexFunc(neededCont.UploadedFiles,
			func(file db.UploadedFile) bool { return file.Path == neededFile.FromPath })]
		err = DockerCli.CopyToContainer(context.TODO(), mainContainerResponse.ID,
			"/neededFiles/", bytes.NewReader(uploadedFile.Bytes), types.CopyToContainerOptions{})
		if err != nil {
			log.Println(err)
		}
	}

	if err := DockerCli.ContainerStart(context.TODO(), mainContainerResponse.ID,
		types.ContainerStartOptions{}); err != nil {
		// Never expect docker to error
		panic(err)
	}

	var logStringBuilder strings.Builder

	response, err := DockerCli.ContainerAttach(context.TODO(), mainContainerResponse.ID, types.ContainerAttachOptions{
		Stream: true, Stdout: true, Logs: true, Stderr: true,
	})
	if err != nil {
		// Never expect docker to error
		panic(err)
	}

	defer response.Close()

	Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d.container.%d.create", build.ID, cont.Id), "")

test:
	for {
		p := make([]byte, 1)
		_, err := response.Reader.Read(p)
		response.Reader.Discard(3)
		var length uint32
		binary.Read(response.Reader, binary.BigEndian, &length)
		if err != nil {
			if err == io.EOF {
				break test
			}
			log.Println(err)
			os.Exit(1)
		}
		p = make([]byte, length)
		n, _ := response.Reader.Read(p)
		logStringBuilder.Write(p[:n])
		err = Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d.container.%d.log", build.ID, cont.Id), p[:n]).Err()
		if err != nil {
			panic(err)
		}
	}

	t, err := DockerCli.ContainerInspect(context.TODO(), mainContainerResponse.ID)
	if err != nil {
		// Never expect docker to error
		panic(err)
	}

	err = Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d.container.%d.die", build.ID, cont.Id), t.State.ExitCode).Err()
	if err != nil {
		panic(err)
	}

	if logStringBuilder.Len() > 100000 {
		db.Db.Model(&cont).Updates(db.Container{Log: logStringBuilder.String()[0:99900] +
			"\nTruncated due to length over 100k chars", Code: &t.State.ExitCode})
	} else {
		db.Db.Model(&cont).Updates(db.Container{Log: logStringBuilder.String(), Code: &t.State.ExitCode})
	}
	for _, file := range cont.UploadedFiles {
		reader, _, err := DockerCli.CopyFromContainer(context.TODO(), mainContainerResponse.ID, file.Path)
		if err != nil {
			log.Println(err)
			db.Db.Model(&cont).Update("log", fmt.Sprintf(
				"%s\nFailed to upload file (%s)", cont.Log, file.Path))
			return
		}
		defer reader.Close()
		bytes, err := io.ReadAll(reader)
		if err != nil {
			break
		}
		db.Db.Model(&file).Update("bytes", bytes)
	}
	DockerCli.ContainerRemove(context.TODO(), mainContainerResponse.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	if cont.ServiceImage != nil {
		DockerCli.ContainerRemove(context.TODO(), serviceContainerResponse.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}
	DockerCli.NetworkRemove(context.TODO(), networkResp.ID)

	if t.State.ExitCode == 0 {
		var edges []db.ContainerGraphEdge
		db.Db.Where(db.ContainerGraphEdge{FromID: uint(cont.Id)}, "FromID").Preload(
			"To.EdgesToward.From").Preload("To.NeededFiles").Preload("To.UploadedFiles").Find(&edges)
		for _, edge := range edges {
			for index, from := range edge.To.EdgesToward {
				if from.From.Code == nil || *from.From.Code != 0 {
					break
				}
				if index == len(edge.To.EdgesToward)-1 {
					wg.Add(1)
					go BuildContainer(repoUrl, buildID, edge.To, organizationId, wg, failed)
				}
			}
		}
	} else {
		*failed = true
	}
}
