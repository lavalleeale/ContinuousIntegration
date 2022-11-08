package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func StartBuild(c *gin.Context) {
	var data struct {
		Command string `form:"command"`
	}
	var containers []struct {
		ID      string `json:"repo_id"`
		Command string `json:"command"`
	}

	c.ShouldBind(&data)

	binding.JSON.BindBody([]byte(data.Command), &containers)

	var repo db.Repo

	db.Db.Find(context.TODO(), &repo, rel.Eq("id", c.Param("repoId")))

	var user db.User

	lib.GetUser(c, &user)

	if user.OrganizationID != repo.OrganizationID {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var build = db.Build{RepoID: repo.ID, Containers: []db.Container{}}

	for _, container := range containers {
		build.Containers = append(build.Containers, db.Container{Name: container.ID, Command: container.Command})
	}

	err := db.Db.Insert(context.TODO(), &build)

	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	for _, container := range build.Containers {
		lib.StartBuild(repo.Url, build.ID, container)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/build/%d", build.ID))
}
