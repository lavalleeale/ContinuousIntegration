package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Login(c *gin.Context) {
	var dat struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	err := c.ShouldBind(&dat)
	if err != nil {
		log.Print("Failed to Unmarshal JSON")
		return
	}

	var user = db.User{Username: dat.Username}
	tx := db.Db.First(&user)

	log.Println(tx.Error)
	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		bytes, err := bcrypt.GenerateFromPassword([]byte(dat.Password), 10)
		if err != nil {
			log.Print(err)
			return
		}
		user = db.User{Username: dat.Username, Password: string(bytes)}
		err = db.Db.Create(&db.Organization{Users: []db.User{user}}).Error
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
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		c.HTML(http.StatusInternalServerError, "login", gin.H{
			"error": "Server Error",
		})
		log.Print("Error Creating Token")
		return
	}

	c.SetCookie(
		"token",
		tokenString,
		2*60*60,
		"/",
		os.Getenv("DOMAIN"),
		os.Getenv("APP_ENV") == "PRODUCTION",
		false,
	)

	c.Redirect(http.StatusFound, "/")
}
