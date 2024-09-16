package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	sessionseal "github.com/lavalleeale/SessionSeal"
	"github.com/minio/minio-go/v7"
)

func finish(cont *db.Container, containers []container.CreateResponse,
	network types.NetworkCreateResponse, code int, currentLog string, mainContainer *container.CreateResponse,
) {
	for _, container := range containers {
		if container.ID != "" {
			DockerCli.ContainerRemove(context.TODO(), container.ID, types.ContainerRemoveOptions{
				Force: true,
			})
		}
	}
	if mainContainer.ID != "" {
		DockerCli.ContainerRemove(context.TODO(), mainContainer.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}
	DockerCli.NetworkRemove(context.TODO(), network.ID)
	if len(currentLog) > 100000 {
		db.Db.Model(&cont).Updates(db.Container{Log: strings.ToValidUTF8(currentLog[0:99900]+
			"\nTruncated due to length over 100k chars", ""), Code: &code})
	} else {
		db.Db.Model(&cont).Updates(db.Container{Log: strings.ToValidUTF8(currentLog, ""), Code: &code})
	}
	err := Rdb.Publish(context.TODO(), fmt.Sprintf(
		"build.%d.container.%s.die", cont.BuildID, cont.Name), code).Err()
	if err != nil {
		panic(err)
	}
}

func BuildContainer(repoUrl string, cont db.Container, organizationId string, wg *sync.WaitGroup, failed *bool) {
	if cont.Persist == nil {
		defer wg.Done()
	}
	networkResp, err := DockerCli.NetworkCreate(context.Background(),
		fmt.Sprintf("%d:%s", cont.BuildID, cont.Name), types.NetworkCreate{
			Driver: "bridge",
		})
	if err != nil {
		log.Fatal(err)
	}

	serviceContainerResponses := make([]container.CreateResponse, len(cont.ServiceContainers))

	var logStringBuilder strings.Builder

	for i, v := range cont.ServiceContainers {
		// Create a container
		var givenCmd []string
		if v.Command == nil {
			givenCmd = nil
		} else {
			givenCmd = []string{"bash", "-c", *v.Command}
		}
		serivceEnv := make([]string, 0)
		if v.Environment != nil {
			serivceEnv = append(serivceEnv, strings.Split(*v.Environment, ",")...)
		}
		response, err := DockerCli.ContainerCreate(context.Background(), &container.Config{
			Image:  v.Image,
			Cmd:    givenCmd,
			Env:    serivceEnv,
			Labels: map[string]string{"buildId": fmt.Sprint(cont.BuildID)},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", v.Healthcheck},
				StartPeriod: time.Second * 30,
				Timeout:     time.Second * 15,
				Interval:    time.Second * 5,
				Retries:     5,
			},
		}, nil, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkResp.ID: {
					Aliases: []string{v.Name},
				},
			},
		}, nil, "")
		if err != nil {
			// Never expect docker to error
			panic(err)
		}

		if err := DockerCli.ContainerStart(context.TODO(), response.ID,
			types.ContainerStartOptions{}); err != nil {
			// Never expect docker to error
			panic(err)
		}

		ctx, close := context.WithTimeout(context.TODO(), time.Minute)

		msgs, errs := DockerCli.Events(ctx, types.EventsOptions{
			Filters: filters.NewArgs(filters.Arg("container", response.ID), filters.Arg("event", "health_status")),
		})
	serviceContainerWaiter:
		for {
			select {
			case err := <-errs:
				close()
				log.Println(err)
				logStringBuilder.WriteString(fmt.Sprintf(
					"Service container %s failed to be healthy", v.Name))
				if errors.Is(err, context.DeadlineExceeded) {
					finish(&cont, serviceContainerResponses,
						networkResp, 255, logStringBuilder.String(), &container.CreateResponse{})
					*failed = true
					return
				}
			case msg := <-msgs:
				if msg.Action == "health_status: healthy" {
					close()
					break serviceContainerWaiter
				}
			}
		}
		serviceContainerResponses[i] = response
	}

	var dockerAuth struct {
		OrganizationID string `json:"organizationId"`
	}

	dockerAuth.OrganizationID = organizationId

	dockerAuthData, _ := json.Marshal(dockerAuth)

	mainEnv := []string{
		"PORT=80",
		"DOCKER_USER=token", fmt.Sprintf("DOCKER_PASS=%s",
			sessionseal.Seal(os.Getenv("JWT_SECRET"), dockerAuthData)),
		fmt.Sprintf("REGISTRY=%s", os.Getenv("REGISTRY_URL")),
	}
	if cont.Environment != nil {
		mainEnv = append(mainEnv, strings.Split(*cont.Environment, ",")...)
	}
	mainContainerResponse, err := DockerCli.ContainerCreate(context.TODO(), &container.Config{
		Image:  cont.Image,
		Cmd:    []string{"bash", "-c", "sleep 1 && " + cont.Command},
		Env:    mainEnv,
		Tty:    false,
		Labels: map[string]string{"buildId": strconv.FormatUint(uint64(cont.BuildID), 10), "containerId": cont.Name},
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}, &container.HostConfig{
		Runtime: os.Getenv("RUNTIME"),
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{},
			},
		},
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

	for _, neededFile := range cont.FilesNeeded {
		object, err := MinioClient.GetObject(context.TODO(), BucketName,
			neededFile.ID.String(), minio.GetObjectOptions{})
		if err != nil {
			log.Println(err)
			continue
		}
		defer object.Close()

		err = DockerCli.CopyToContainer(context.TODO(), mainContainerResponse.ID,
			"/neededFiles/", object, types.CopyToContainerOptions{})
		if err != nil {
			log.Println(err)
		}
	}

	if err := DockerCli.ContainerStart(context.TODO(), mainContainerResponse.ID,
		types.ContainerStartOptions{}); err != nil {
		// Never expect docker to error
		panic(err)
	}

	db.Db.Model(&cont).Update("log", "Started!\n")
	var allowedTime time.Duration
	if cont.Persist != nil {
		allowedTime = time.Hour * 24 * 7

		inspectionData, err := DockerCli.ContainerInspect(context.TODO(), mainContainerResponse.ID)
		if err != nil {
			// Never expect docker to error
			panic(err)
		}
		Rdb.Set(context.TODO(), *cont.Persist, inspectionData.NetworkSettings.Ports["80/tcp"][0].HostPort, allowedTime)
	} else {
		allowedTime = time.Minute * 30
	}

	ctx, close := context.WithTimeout(context.TODO(), allowedTime)

	response, err := DockerCli.ContainerAttach(ctx, mainContainerResponse.ID, types.ContainerAttachOptions{
		Stream: true, Stdout: true, Logs: true, Stderr: true,
	})
	if err != nil {
		// Never expect docker to error
		panic(err)
	}

	go func() {
		<-ctx.Done()
		response.Close()
	}()

	Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d.container.%s.create", cont.BuildID, cont.Name), "")

	output, errCh := ReadAttach(response.Reader)

readLoop:
	for {
		select {
		case err := <-errCh:
			if err == io.EOF {
				break readLoop
			} else if errors.Is(err, net.ErrClosed) {
				close()
				logStringBuilder.WriteString("Container dealine exceeded")
				finish(&cont, serviceContainerResponses, networkResp, 255, logStringBuilder.String(),
					&mainContainerResponse)
				*failed = true
				return
			}
			panic(err)
		case outputData := <-output:
			logStringBuilder.Write(bytes.ReplaceAll(outputData, []byte{0}, []byte{}))
			err = Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d.container.%s.log",
				cont.BuildID, cont.Name), outputData).Err()
			if err != nil {
				panic(err)
			}
		}
	}
	close()

	t, err := DockerCli.ContainerInspect(context.TODO(), mainContainerResponse.ID)
	if err != nil {
		// Never expect docker to error
		panic(err)
	}

	for _, file := range cont.FilesUploaded {
		reader, _, err := DockerCli.CopyFromContainer(context.TODO(), mainContainerResponse.ID, file.Path)
		if err != nil {
			logStringBuilder.WriteString(fmt.Sprintf("Failed to upload file (%s)", file.Path))
			finish(&cont, serviceContainerResponses, networkResp, 255, logStringBuilder.String(),
				&mainContainerResponse)
			*failed = true
			return
		}
		defer reader.Close()
		buf := new(bytes.Buffer)
		n, err := buf.ReadFrom(reader)
		if err != nil {
			logStringBuilder.WriteString(fmt.Sprintf("Failed to upload file (%s)", file.Path))
			finish(&cont, serviceContainerResponses, networkResp, 255, logStringBuilder.String(),
				&mainContainerResponse)
			*failed = true
			return
		}
		_, err = MinioClient.PutObject(context.TODO(), BucketName, file.ID.String(), buf,
			n, minio.PutObjectOptions{ContentType: "application/x-tar"})
		if err != nil {
			logStringBuilder.WriteString(fmt.Sprintf("Failed to upload file (%s)", file.Path))
			finish(&cont, serviceContainerResponses, networkResp, 255, logStringBuilder.String(),
				&mainContainerResponse)
			*failed = true
			return
		}
	}

	finish(&cont, serviceContainerResponses, networkResp, t.State.ExitCode, logStringBuilder.String(),
		&mainContainerResponse)

	if t.State.ExitCode == 0 {
		var edges []db.ContainerGraphEdge
		db.Db.Where("from_name = ? AND build_id = ?", cont.Name, cont.BuildID).
			Preload("To.EdgesToward.From").
			Preload("To.FilesNeeded").
			Preload("To.FilesUploaded").
			Preload("To.ServiceContainers").
			Find(&edges)
		for _, edge := range edges {
			for index, from := range edge.To.EdgesToward {
				if from.From.Code == nil || *from.From.Code != 0 {
					break
				}
				if index == len(edge.To.EdgesToward)-1 {
					if edge.To.Persist == nil {
						wg.Add(1)
					}
					go BuildContainer(repoUrl, edge.To, organizationId, wg, failed)
				}
			}
		}
	} else {
		*failed = true
	}
}
