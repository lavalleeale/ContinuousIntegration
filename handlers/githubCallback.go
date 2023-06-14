package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
	"github.com/lib/pq"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
	provider "golang.org/x/oauth2/github"
	"gorm.io/gorm"
)

func GithubCallback(c *gin.Context) {
	installId, err := strconv.ParseInt(c.Query("installation_id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	config := oauth2.Config{ClientID: os.Getenv("GITHUB_CLIENT_ID"), ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"), Endpoint: provider.Endpoint}
	token, err := config.Exchange(context.TODO(), c.Query("code"))
	if err != nil {
		// Code Invalid, should never happen unless user is trying to gain access, does not need explanation
		c.Redirect(http.StatusFound, "/")
		return
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.AccessToken},
	)
	tc := oauth2.NewClient(context.TODO(), ts)

	accessTokenClient := github.NewClient(tc)
	user, _, err := accessTokenClient.Apps.ListUserInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		// Should never fail since access token was just obtained so does not need explanation
		log.Println(err)
		c.Redirect(http.StatusFound, "/")
		return
	}
	if slices.IndexFunc(user, func(installation *github.Installation) bool {
		return *installation.ID == installId
	}) != -1 {
		//verified install id is owned by current user
		var user db.User
		if lib.GetUser(c, &user) {
			db.Db.Model(&user).Update("installation_ids", gorm.Expr("installation_ids || ?", pq.Array([]int64{installId})))
			c.Redirect(http.StatusFound, "/addRepoGithub")
		} else {
			lib.SetSession(c, "installId", strconv.FormatInt(installId, 10))
			c.Redirect(http.StatusFound, "/login")
		}
	}
}
