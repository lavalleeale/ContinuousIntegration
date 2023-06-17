package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

func AddRepo(c *gin.Context) {
	var user db.User

	if !lib.GetUser(c, &user) {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var data struct {
		Url string `form:"url"`
	}

	c.ShouldBind(&data)

	repo := db.Repo{Url: data.Url, OrganizationID: user.OrganizationID}

	err := db.Db.Create(&repo).Error
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/repo/%d", repo.ID))
}
