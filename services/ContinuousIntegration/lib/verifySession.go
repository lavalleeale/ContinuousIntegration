package lib

import (
	"encoding/json"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

func GetUser(c *gin.Context, user *db.User) bool {
	session := c.MustGet("session").(map[string]string)
	username, ok := session["username"]
	if ok {
		user.Username = username
		db.Db.First(&user)
		return true
	}
	return false
}

func VerifySession(sessionString string) (map[string]string, error) {
	marshaledData, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), sessionString)
	if err != nil {
		return nil, err
	}
	var data map[string]string
	json.Unmarshal(marshaledData, &data)
	return data, nil
}
