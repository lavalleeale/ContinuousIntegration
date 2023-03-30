package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func StartBuild(c *gin.Context) {
	var data struct {
		Command string `form:"command"`
	}

	var containers []struct {
		ID          string               `json:"id"`
		Steps       []string             `json:"steps"`
		Image       string               `json:"image"`
		Environment *[]map[string]string `json:"environment,omitempty"`
		Service     *struct {
			Steps       *[]string            `json:"steps,omitempty"`
			Environment *[]map[string]string `json:"environment,omitempty"`
			Image       string               `json:"image"`
			Healthcheck string               `json:"healthcheck"`
		} `json:"service,omitempty"`
	}

	c.ShouldBind(&data)

	binding.JSON.BindBody([]byte(data.Command), &containers)

	repoId, err := strconv.Atoi(c.Param("repoId"))

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	repo := db.Repo{ID: repoId}

	db.Db.First(&repo)

	var user db.User

	lib.GetUser(c, &user)

	if user.OrganizationID != repo.OrganizationID {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var build = db.Build{RepoID: repo.ID, Containers: []db.Container{}}

	for _, container := range containers {
		savedContainer := db.Container{
			Name:    container.ID,
			Command: strings.Join(container.Steps, " && "),
			Image:   container.Image,
		}
		if container.Environment != nil {
			var environment = make([]string, 0)
			for _, item := range *container.Environment {
				for k, v := range item {
					environment = append(environment, fmt.Sprintf("%s=%s", k, v))
				}
			}
			environmentString := strings.Join(environment, ",")
			savedContainer.Environment = &environmentString
		}
		if container.Service != nil {
			savedContainer.ServiceImage = &container.Service.Image
			savedContainer.ServiceHealthcheck = &container.Service.Healthcheck
			if container.Service.Steps != nil {
				command := strings.Join(*container.Service.Steps, " && ")
				savedContainer.ServiceCommand = &command
			}
			if container.Service.Environment != nil {
				var environment = make([]string, 0)
				for _, item := range *container.Service.Environment {
					for k, v := range item {
						environment = append(environment, fmt.Sprintf("%s=%s", k, v))
					}
				}
				environmentString := strings.Join(environment, ",")
				savedContainer.ServiceEnvironment = &environmentString
			}
		}
		build.Containers = append(build.Containers, savedContainer)
	}

	err = db.Db.Create(&build).Error

	if err != nil {
		log.Println(err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	for _, container := range build.Containers {
		go lib.StartBuild(repo.Url, build.ID, container)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/build/%d", build.ID))
}
