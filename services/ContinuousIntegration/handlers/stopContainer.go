package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

func StopContainer(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	containerId, err := strconv.Atoi(c.Param("containerId"))
	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	containerData := db.Container{Id: uint(containerId)}

	db.Db.Preload("Build.Repo").First(&containerData)

	if containerData.Build.Repo.OrganizationID == user.OrganizationID {
		filter := filters.NewArgs(
			filters.KeyValuePair{
				Key: "label",
				Value: fmt.Sprintf("buildId=%s",
					c.Param("buildId"),
				),
			},
			filters.KeyValuePair{
				Key:   "label",
				Value: fmt.Sprintf("containerId=%s", c.Param("containerId")),
			},
		)
		containers, err := lib.DockerCli.ContainerList(
			context.TODO(),
			types.ContainerListOptions{
				All:     true,
				Size:    true,
				Filters: filter,
			},
		)
		if err != nil {
			panic(err)
		}
		if (len(containers)) == 0 {
			c.Redirect(http.StatusFound, "/")
			return
		}
		lib.DockerCli.ContainerStop(context.Background(), containers[0].ID, container.StopOptions{})
		c.Redirect(http.StatusFound, "/build/"+c.Param("buildId"))
		return
	}
	c.Redirect(http.StatusFound, "/")
}
