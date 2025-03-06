package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

func InviteUser(c *gin.Context) {
	var data struct {
		Username string `form:"username"`
	}

	c.ShouldBind(&data)

	var user db.User

	if !lib.GetUser(c, &user) {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if len(data.Username) < 5 {
		c.Redirect(http.StatusFound, c.Request.Referer())
		return
	}

	// make sure username not taken
	var existingUser db.User
	db.Db.Where("username = ?", data.Username).First(&existingUser)

	if existingUser.Username != "" {
		c.Redirect(http.StatusFound, c.Request.Referer())
		return
	}

	invitedUser := db.OrganizationInvite{
		Username:       data.Username,
		OrganizationID: user.OrganizationID,
	}

	db.Db.Create(&invitedUser)

	c.HTML(http.StatusOK, "invite", invitedUser)
}
