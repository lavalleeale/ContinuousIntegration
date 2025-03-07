package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	sessionseal "github.com/lavalleeale/SessionSeal"
	"golang.org/x/crypto/bcrypt"
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
	db.Db.Where("username = ?", data.Username).Find(&existingUser)

	if existingUser.Username != "" {
		c.Redirect(http.StatusFound, c.Request.Referer())
		return
	}

	invitedUser := db.OrganizationInvite{
		Username:       data.Username,
		OrganizationID: user.OrganizationID,
	}

	db.Db.Create(&invitedUser)

	var marshalledInvite []byte
	marshalledInvite, err := json.Marshal(invitedUser)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create invite")
		return
	}
	sealed := sessionseal.Seal(os.Getenv("JWT_SECRET"), marshalledInvite)

	c.HTML(http.StatusOK, "invite", gin.H{
		"invite": fmt.Sprintf("%s/acceptInvite?data=%s", os.Getenv("URL"), sealed),
	})
}

func AcceptInvite(c *gin.Context) {
	var form struct {
		Data     string `form:"data"`
		Password string `form:"password"`
	}

	c.ShouldBind(&form)

	if form.Data == "" {
		c.String(http.StatusBadRequest, "Missing data")
		return
	}

	inviteBytes, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), form.Data)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid data")
		return
	}

	var invite db.OrganizationInvite
	err = json.Unmarshal(inviteBytes, &invite)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid data")
		return
	}

	if invite.Username == "" {
		c.String(http.StatusBadRequest, "Invalid data")
		return
	}

	if invite.OrganizationID == "" {
		c.String(http.StatusBadRequest, "Invalid data")
		return
	}

	var user db.User
	db.Db.Where("username = ?", invite.Username).Find(&user)

	if user.Username != "" {
		c.String(http.StatusBadRequest, "User already exists")
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(form.Password), 10)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user = db.User{Username: invite.Username, Password: string(bytes), OrganizationID: invite.OrganizationID}
	db.Db.Create(&user)

	lib.SetSession(c, "username", user.Username)
	c.Redirect(http.StatusFound, "/")
}
