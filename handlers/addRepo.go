package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func AddRepo(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	var data struct {
		Url string `form:"url"`
	}

	c.ShouldBind(&data)

	repo := db.Repo{Url: data.Url, OrganizationID: user.OrganizationID}

	err := db.Db.Create(&repo)

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/repo/%d", repo.ID))
}
