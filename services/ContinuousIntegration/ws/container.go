package ws

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

func HandleContainerWs(c *gin.Context) {
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

	containerId, err := strconv.Atoi(c.Param("containerId"))
	if err != nil {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	container := db.Container{Id: uint(containerId)}
	err = db.Db.Preload("Build.Repo").First(&container).Error

	if err != nil || container.Build.Repo.OrganizationID != user.OrganizationID {
		socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, ""), time.Now().Add(time.Second))
		return
	}

	if len(container.Log) != 0 {
		socket.WriteJSON(gin.H{"type": "log", "log": container.Log})
		return
	}

	pubsub := lib.Rdb.PSubscribe(context.TODO(), fmt.Sprintf("build.%s.container.%s.*",
		c.Param("buildId"), c.Param("containerId")))
	defer pubsub.Close()

	ch := pubsub.Channel()
outer:
	for msg := range ch {
		switch strings.Split(msg.Channel, ".")[4] {
		case "log":
			socket.WriteJSON(gin.H{"type": "log", "log": msg.Payload})
		case "die":
			socket.WriteJSON(gin.H{"type": "code", "code": msg.Payload})
			break outer
		}
	}
}
