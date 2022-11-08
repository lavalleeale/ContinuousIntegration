package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleBuildWs(c *gin.Context) {
	var wg sync.WaitGroup
	var user db.User

	socket, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	username, err := lib.VerifyJwtString(c.Query("token"))
	db.Db.Find(context.TODO(), &user, rel.Eq("username", username))

	log.Println(username)

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	numId, err := strconv.ParseInt(c.Param("buildId"), 10, 64)

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	var build db.Build
	err = db.Db.Find(context.TODO(), &build, rel.Eq("id", numId))
	db.Db.Preload(context.TODO(), &build, "repo")

	if err != nil || build.Repo.OrganizationID != user.OrganizationID {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	filters := filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("buildId=%s", c.Param("buildId"))})
	containers, err := lib.Cli.ContainerList(context.TODO(), types.ContainerListOptions{All: true, Size: true, Filters: filters})

	if err != nil || len(containers) == 0 {
		socket.WriteJSON(gin.H{"error": "build finished"})
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	for _, container := range containers {
		wg.Add(1)
		go AttachContainers(socket, container.ID, container.Labels["containerId"], &wg)
	}

	wg.Wait()
	socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
}

func AttachContainers(c *websocket.Conn, containerId string, containerLabel string, wg *sync.WaitGroup) {
	defer wg.Done()
	statusCh, errCh := lib.Cli.ContainerWait(context.TODO(), containerId, container.WaitConditionNextExit)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case comp := <-statusCh:
		c.WriteJSON(gin.H{"id": containerLabel, "code": comp.StatusCode})
	}
}
