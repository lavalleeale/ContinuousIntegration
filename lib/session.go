package lib

import (
	"encoding/json"
	"os"

	"github.com/gin-gonic/gin"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

func Session(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		userData, err := VerifySession(cookie.Value)
		if err == nil {
			c.Set("session", userData)
		} else {
			c.Set("session", map[string]string{})
		}
	} else {
		c.Set("session", map[string]string{})
	}

	// before request

	c.Next()

	var sessionData = c.MustGet("session").(map[string]string)
	session, err := json.Marshal(sessionData)
	if err != nil {
		panic(err)
	}
	c.SetCookie(
		"session",
		sessionseal.Seal(os.Getenv("JWT_SECRET"), session),
		2*60*60,
		"/",
		os.Getenv("DOMAIN"),
		os.Getenv("APP_ENV") == "PRODUCTION",
		false,
	)
}
