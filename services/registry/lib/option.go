package lib

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type Option struct {
	issuer, typ, name, account, service string
	actions                             []string // requested actions
}

func CreateTokenOption(c *gin.Context) *Option {
	opt := &Option{}
	user, _, _ := c.Request.BasicAuth()

	opt.service = c.Query("service")
	opt.account = user
	opt.issuer = "ci-issuer" // issuer value must match the value configured via docker-compose

	parts := strings.Split(c.Query("scope"), ":")
	if len(parts) > 0 {
		opt.typ = parts[0] // repository
	}
	if len(parts) > 1 {
		opt.name = parts[1] // foo/repoName
	}
	if len(parts) > 2 {
		opt.actions = strings.Split(parts[2], ",") // requested actions
	}
	return opt
}
