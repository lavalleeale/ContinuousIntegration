package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
	"golang.org/x/exp/slices"
)

func AddRepoGithub(c *gin.Context) {
	var data struct {
		RepoId    int64 `form:"repoId"`
		InstallId int64 `form:"installId"`
	}
	c.ShouldBind(&data)
	var user db.User
	if lib.GetUser(c, &user) {
		if slices.Contains(user.InstallationIds, data.InstallId) {
			client := github.NewClient(&http.Client{
				Transport: ghinstallation.NewFromAppsTransport(lib.Itr, data.InstallId),
				Timeout:   time.Second * 30,
			})
			installRepos, _, err := client.Apps.ListRepos(context.TODO(), &github.ListOptions{})
			if err != nil {
				c.Redirect(http.StatusFound, "/")
				return
			}
			repoIndex := slices.IndexFunc(installRepos, func(repo *github.Repository) bool { return *repo.ID == data.RepoId })
			if repoIndex != -1 {
				repo := db.Repo{Url: *installRepos[repoIndex].HTMLURL, InstallationId: &data.InstallId, GithubRepoId: &data.RepoId, OrganizationID: user.OrganizationID}
				db.Db.Create(&repo)
				c.Redirect(http.StatusFound, fmt.Sprintf("/repo/%d", repo.ID))
				return
			}
		}
	}
	c.Redirect(http.StatusFound, "/")
}
