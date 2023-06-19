package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/auth"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

type AuthData struct {
	OrganizationID string `json:"organizationId"`
}

func Authenticate(c *gin.Context, opt *Option) (AuthData, error) {
	username, password, ok := c.Request.BasicAuth()
	if !ok {
		return AuthData{}, fmt.Errorf("auth credentials not found")
	}
	if username != opt.account {
		return AuthData{}, fmt.Errorf("invalid username")
	}

	if username == "token" {
		marshaledData, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), password)
		if err != nil {
			return AuthData{}, fmt.Errorf("invalid password")
		}

		var data AuthData

		err = json.Unmarshal(marshaledData, &data)

		if err != nil {
			return AuthData{}, fmt.Errorf("invalid json")
		}

		return data, nil
	} else {
		user, err := auth.Login(username, password, false)
		if err != nil {
			return AuthData{}, err
		}
		return AuthData{OrganizationID: user.OrganizationID}, nil
	}
}

func Authorize(opt *Option, data AuthData) []string {
	if data.OrganizationID == "root" {
		return []string{"pull", "push"}
	}
	if strings.Split(opt.name, "/")[0] == data.OrganizationID {
		return []string{"pull", "push"}
	}
	// unauthorized, no permission is granted
	return []string{}
}
