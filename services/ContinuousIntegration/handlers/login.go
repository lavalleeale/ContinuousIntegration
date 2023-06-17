package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
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

	user := db.User{Username: dat.Username}
	tx := db.Db.First(&user)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		bytes, err := bcrypt.GenerateFromPassword([]byte(dat.Password), 10)
		if err != nil {
			log.Print(err)
			return
		}
		user = db.User{Username: dat.Username, Password: string(bytes)}
		if waitingForInstall {
			id, err := strconv.ParseInt(installId, 10, 64)
			if err != nil {
				// This will never fail since the user cannot alter their own session
				panic(err)
			}
			user.InstallationIds = pq.Int64Array{id}
		}
		err = db.Db.Create(&db.Organization{Users: []db.User{user}, ID: user.Username}).Error
		if err != nil {
			c.HTML(http.StatusInternalServerError, "login", gin.H{
				"error": "Failed to create user",
			})
			return
		}
	} else {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dat.Password))
		if err != nil {
			c.HTML(http.StatusUnauthorized, "login", gin.H{
				"error": "Invalid Password",
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
	}

	lib.SetSession(c, "username", user.Username)

	if waitingForInstall {
		c.Redirect(http.StatusFound, "/addRepoGithub")
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}
