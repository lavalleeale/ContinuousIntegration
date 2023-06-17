package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

func DeleteRepo(c *gin.Context) {
	var user db.User

	if !lib.GetUser(c, &user) {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var data struct {
		Id string `form:"id"`
	}

	c.ShouldBind(&data)

	repoId, err := strconv.ParseInt(data.Id, 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	repo := db.Repo{ID: uint(repoId)}

	err = db.Db.Delete(&repo).Error
	c.Redirect(http.StatusFound, "/")
}
