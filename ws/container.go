package ws

import (
	"context"
	"encoding/binary"
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

	go AttachContainer(socket, c.Param("buildId"), c.Param("containerId"))
}

func AttachContainer(c *websocket.Conn, BuildID string, ContainerID string) {
	filters := filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("buildId=%s", BuildID)}, filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("containerId=%s", ContainerID)})
	containers, err := lib.Cli.ContainerList(context.TODO(), types.ContainerListOptions{All: true, Size: true, Filters: filters})

	if err != nil || len(containers) == 0 {
		c.WriteJSON(gin.H{"error": "build not found"})
		return
	}

	response, err := lib.Cli.ContainerAttach(context.TODO(), containers[0].ID, types.ContainerAttachOptions{Stream: true, Stdout: true, Logs: true, Stderr: true})

	if err != nil {
		c.WriteJSON(gin.H{"error": "build not found"})
		return
	}

	defer response.Close()

	for {
		p := make([]byte, 1)
		_, err := response.Reader.Read(p)
		response.Reader.Discard(3)
		var length int32
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
		c.WriteJSON(gin.H{"type": "log", "log": string(p[:n])})
	}

	statusCh, errCh := lib.Cli.ContainerWait(context.TODO(), containers[0].ID, container.WaitConditionNextExit)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case comp := <-statusCh:
		c.WriteJSON(gin.H{"type": "code", "code": comp.StatusCode})
	}

	c.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second))
}
