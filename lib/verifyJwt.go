package lib

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
)

func GetUser(c *gin.Context, user *db.User) bool {
	name, valid := VerifyJwt(c)
	if !valid {
		return false
	}
	db.Db.Find(context.TODO(), user, rel.Eq("username", name))
	return true
}

func VerifyJwt(c *gin.Context) (string, bool) {
	tokenString, err := c.Request.Cookie("token")
	if err != nil || tokenString == nil {
		return "", false
	}
	token, err := VerifyJwtString(tokenString.Value)
	if err != nil {
		return "", false
	}
	return token, true
}

func VerifyJwtString(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return "", errors.New("invalid jwt")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return fmt.Sprint(claims["username"]), nil
	} else {
		return "", errors.New("incorrect claims")
	}
}
