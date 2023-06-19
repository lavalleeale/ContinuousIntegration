package lib

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
)

func Handler(m *autocert.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		opt := CreateTokenOption(c)

		data, err := Authenticate(c, opt)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		actions := Authorize(opt, data)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("server error"))
			return
		}

		cert, err := m.GetCertificate(&tls.ClientHelloInfo{
			ServerName:      os.Getenv("HOST"),
			SupportedProtos: []string{},
		})
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("server error"))
			return
		}
		tk, err := CreateToken(opt, actions, cert)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("server error"))
			return
		}
		c.JSON(http.StatusOK, tk)
	}
}
