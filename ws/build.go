package ws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
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

	cookie, err := c.Request.Cookie("token")

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	username, err := lib.VerifyJwtString(cookie.Value)
	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}
	user := db.User{Username: username}

	err = db.Db.First(&user).Error

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	numId, err := strconv.Atoi(c.Param("buildId"))

	if err != nil {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	build := db.Build{ID: numId, Repo: db.Repo{OrganizationID: user.OrganizationID}}
	err = db.Db.Preload("Containers").Preload("Repo").Where(&build, "id", "Repo.OrganizationID").First(&build).Error

	if err != nil || build.Repo.OrganizationID != user.OrganizationID {
		socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	left := 0

	for index, cont := range build.Containers {
		if cont.Code == nil {
			left++
		} else if left == 0 && index == len(build.Containers)-1 {
			socket.WriteJSON(gin.H{"error": "build finished"})
			socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			return
		}
	}

	filters := filters.NewArgs(
		filters.KeyValuePair{
			Key: "label",
			Value: fmt.Sprintf(
				"buildId=%s",
				c.Param("buildId"),
			),
		},
		filters.KeyValuePair{
			Key:   "event",
			Value: "die",
		},
		filters.KeyValuePair{
			Key:   "event",
			Value: "create",
		},
	)
	ctx, close := context.WithTimeout(context.TODO(), time.Minute*30)

	msgs, errs := lib.Cli.Events(ctx, types.EventsOptions{
		Filters: filters,
	})

outer:
	for {
		select {
		case err := <-errs:
			log.Println(err)
			close()
			if errors.Is(err, context.DeadlineExceeded) {
				socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
				return
			}
		case msg := <-msgs:
			if msg.Action == "die" {
				socket.WriteJSON(gin.H{"type": "die", "id": msg.Actor.Attributes["containerId"], "code": msg.Actor.Attributes["exitCode"]})
				left--
				if left == 0 {
					close()
					break outer
				}
			} else {
				socket.WriteJSON(gin.H{"type": "create", "id": msg.Actor.Attributes["containerId"]})
			}
		}
	}
	socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
}
