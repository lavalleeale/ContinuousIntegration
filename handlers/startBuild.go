package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation"
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

	if !lib.GetUser(c, &user) {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if user.OrganizationID != repo.OrganizationID {
		c.Redirect(http.StatusFound, "/")
		return
	}

	authData := []string{}
	if repo.InstallationId != nil {
		transport := ghinstallation.NewFromAppsTransport(lib.Itr, *repo.InstallationId)
		token, err := transport.Token(context.TODO())
		if err != nil {
			// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
			panic(err)
		}
		authData = []string{"x-access-token", token}
	}

	build, err := lib.StartBuild(repo, buildData, authData, func(id uint, failed bool) {
		if failed {
			db.Db.Model(db.Build{ID: id}).Update("status", "failed")
		} else {
			db.Db.Model(db.Build{ID: id}).Update("status", "succeeded")
		}
	})

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/build/%d", build.ID))
}
