package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/auth"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func Login(c *gin.Context) {
	session := c.MustGet("session").(map[string]string)
	installId, waitingForInstall := session["installId"]
	var dat struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	if dat.Username == "root" {
		return
	}

	err := c.ShouldBind(&dat)
	if err != nil {
		log.Print("Failed to Unmarshal JSON")
		return
	}

	user, err := auth.Login(dat.Username, dat.Password, true)

	if err != nil {
		c.HTML(http.StatusInternalServerError, "login", gin.H{
			"error": err,
		})
		return
	}

	if waitingForInstall {
		id, err := strconv.ParseInt(installId, 10, 64)
		if err != nil {
			// This will never fail since the user cannot alter their own session
			panic(err)
		}

		db.Db.Model(&user).Update("installation_ids",
			gorm.Expr("installation_ids || ?", pq.Array([]int64{id})))
	}

	lib.SetSession(c, "username", user.Username)

	if waitingForInstall {
		c.Redirect(http.StatusFound, "/addRepoGithub")
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}
