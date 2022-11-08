package handlers

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	ctx := context.TODO()
	var dat struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	err := c.ShouldBind(&dat)
	if err != nil {
		log.Print("Failed to Unmarshal JSON")
		return
	}

	var user db.User
	db.Db.Find(ctx, &user, rel.Eq("username", dat.Username))

	if user.ID == 0 {
		bytes, err := bcrypt.GenerateFromPassword([]byte(dat.Password), 10)
		if err != nil {
			log.Print(err)
			return
		}
		user = db.User{Username: dat.Username, Password: string(bytes)}
		db.Db.Insert(ctx, &db.Organization{Users: []db.User{user}})
	} else {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dat.Password))
		if err != nil {
			c.HTML(http.StatusUnauthorized, "login", gin.H{
				"error": "Invalid Password",
			})
			return
		}
	}

	db.Db.Preload(context.TODO(), &user, "organization")
	db.Db.Preload(context.TODO(), &user, "organization.repos")

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
