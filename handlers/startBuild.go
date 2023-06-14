package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func StartBuild(c *gin.Context) {
	var data struct {
		Command string `form:"command"`
	}

	var buildData lib.BuildData

	c.ShouldBind(&data)

	binding.JSON.BindBody([]byte(data.Command), &buildData)

	repoId, err := strconv.Atoi(c.Param("repoId"))

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	repo := db.Repo{ID: uint(repoId)}

	db.Db.First(&repo)

	var user db.User

	lib.GetUser(c, &user)

	if user.OrganizationID != repo.OrganizationID {
		c.Redirect(http.StatusFound, "/")
		return
	}

	build, err := lib.StartBuild(repo, buildData, []string{}, nil)

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/build/%d", build.ID))
}
