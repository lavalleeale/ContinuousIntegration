package lib

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

func Session(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		sessionData, err := VerifySession(cookie.Value)
		if err == nil {
			c.Set("session", sessionData)
		} else {
			c.Set("session", map[string]string{})
		}
	} else {
		c.Set("session", map[string]string{})
	}
}

func SetSession(c *gin.Context, key string, value string) {
	sessionData := c.MustGet("session").(map[string]string)
	sessionData[key] = value
	session, err := json.Marshal(sessionData)
	if err != nil {
		// We have created map so marshalling it should never fail
		panic(err)
	}
	c.SetSameSite(http.SameSiteLaxMode)
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
