package ws

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/gin-gonic/gin"
	"github.com/go-rel/rel"
	"github.com/gorilla/websocket"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func HandleContainerWs(c *gin.Context) {
	socket, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	var user db.User

	username, err := lib.VerifyJwtString(c.Query("token"))
	db.Db.Find(context.TODO(), &user, rel.Eq("username", username))

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	numId, err := strconv.ParseInt(c.Param("buildId"), 10, 64)

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	var container db.Container
	err = db.Db.Find(context.TODO(), &container, rel.And(rel.Eq("build_id", numId), rel.Eq("id", c.Param("containerId"))))
	db.Db.Preload(context.TODO(), &container, "build")
	db.Db.Preload(context.TODO(), &container, "build.repo")

	if err != nil || container.Build.Repo.OrganizationID != user.OrganizationID {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	if len(container.Log) != 0 {
		socket.WriteJSON(gin.H{"type": "log", "log": container.Log})
		return
	}

	go AttachContainer(socket, c.Param("buildId"), c.Param("containerId"))
}

func AttachContainer(socket *websocket.Conn, BuildID string, ContainerID string) {
	filter := filters.NewArgs(
		filters.KeyValuePair{
			Key: "label",
			Value: fmt.Sprintf("buildId=%s",
				BuildID,
			),
		},
		filters.KeyValuePair{
			Key:   "label",
			Value: fmt.Sprintf("containerId=%s", ContainerID),
		},
	)
	containers, err := lib.Cli.ContainerList(
		context.TODO(),
		types.ContainerListOptions{
			All:     true,
			Size:    true,
			Filters: filter,
		},
	)

	var containerId string

	if err != nil {
		log.Println(err)
		socket.WriteJSON(gin.H{"error": "build not found"})
		return
	} else if len(containers) == 0 {
		socket.WriteJSON(gin.H{"type": "log", "log": "Waiting for container to start\n"})
		socket.WriteJSON(gin.H{"type": "code", "code": "Waiting"})
		ctx, close := context.WithTimeout(context.TODO(), time.Second*90)

		filter.Add("event", "start")
		msgs, errs := lib.Cli.Events(ctx, types.EventsOptions{Filters: filter})

	outer:
		for {
			select {
			case err := <-errs:
				log.Println(err)
				close()
				if errors.Is(err, context.DeadlineExceeded) {
					socket.WriteJSON(gin.H{"type": "log", "log": "Container failed to start in time, please refresh"})
					socket.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second))
				}
				return
			case msg := <-msgs:
				containerId = msg.Actor.ID
				close()
				break outer
			}
		}
	} else {
		containerId = containers[0].ID
	}

	response, err := lib.Cli.ContainerAttach(context.TODO(), containerId, types.ContainerAttachOptions{Stream: true, Stdout: true, Logs: true, Stderr: true})

	if err != nil {
		log.Println(err)
		socket.WriteJSON(gin.H{"error": "attach failed"})
		return
	}

	defer response.Close()

	for {
		p := make([]byte, 1)
		_, err := response.Reader.Read(p)
		response.Reader.Discard(3)
		var length uint32
		binary.Read(response.Reader, binary.BigEndian, &length)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println(err)
			os.Exit(1)
		}
		p = make([]byte, length)
		n, _ := response.Reader.Read(p)
		socket.WriteJSON(gin.H{"type": "log", "log": string(p[:n])})
	}

	statusCh, errCh := lib.Cli.ContainerWait(context.TODO(), containerId, container.WaitConditionNextExit)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case comp := <-statusCh:
		socket.WriteJSON(gin.H{"type": "code", "code": comp.StatusCode})
	}

	socket.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second))
}
