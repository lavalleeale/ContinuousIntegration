package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleBuildWs(c *gin.Context) {
	socket, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	session := c.MustGet("session").(map[string]string)

	username, ok := session["username"]

	if !ok {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}
	user := db.User{Username: username}

	err = db.Db.First(&user).Error

	if err != nil {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	numId, err := strconv.Atoi(c.Param("buildId"))
	if err != nil {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	pubsub := lib.Rdb.PSubscribe(context.TODO(), fmt.Sprintf("build.%d.*", numId))
	defer pubsub.Close()
	ch := pubsub.Channel()
	build := db.Build{ID: uint(numId), Repo: db.Repo{OrganizationID: user.OrganizationID}}
	err = db.Db.Preload("Containers").Preload("Repo").First(&build).Error

	if err != nil || build.Repo.OrganizationID != user.OrganizationID {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	left := 0

	for _, cont := range build.Containers {
		if cont.Code == nil {
			left++
		} else {
			socket.WriteJSON(gin.H{
				"type": "die", "id": cont.Id,
				"code": strconv.FormatInt(int64(*cont.Code), 10),
			})
		}
	}

	if left == 0 {
		socket.WriteJSON(gin.H{"error": "build finished"})
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	containers, err := lib.DockerCli.ContainerList(context.TODO(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: fmt.Sprintf("buildId=%s", c.Param("buildId")),
		}),
	})
	if err != nil {
		// Never expect docker to error
		panic(err)
	}
	for _, cont := range containers {
		socket.WriteJSON(gin.H{"type": "create", "id": cont.Labels["containerId"]})
	}

outer:
	for msg := range ch {
		channel := strings.Split(msg.Channel, ".")
		switch channel[4] {
		case "die":
			left--
			socket.WriteJSON(gin.H{
				"type": "die", "id": channel[3],
				"code": msg.Payload,
			})
			if left == 0 {
				break outer
			}
		case "log":
		default:
			socket.WriteJSON(gin.H{"type": channel[4], "id": channel[3]})
		}
	}

	socket.WriteJSON(gin.H{"type": "finished"})

	socket.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
}
