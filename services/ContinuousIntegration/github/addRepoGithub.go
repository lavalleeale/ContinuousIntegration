package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
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
				log.Println(err)
				c.Redirect(http.StatusFound, "/")
				return
			}
			repoIndex := slices.IndexFunc(installRepos,
				func(repo *github.Repository) bool { return *repo.ID == data.RepoId })
			if repoIndex != -1 {
				repo := db.Repo{
					Url:            *installRepos[repoIndex].HTMLURL,
					InstallationId: &data.InstallId, GithubRepoId: &data.RepoId, OrganizationID: user.OrganizationID,
				}
				db.Db.Create(&repo)
				c.Redirect(http.StatusFound, fmt.Sprintf("/repo/%d", repo.ID))
				return
			}
		}
	}
	c.Redirect(http.StatusFound, "/")
}

func AddRepoGithhubPage(c *gin.Context) {
	var user db.User
	repos := map[int64][]*github.Repository{}
	if lib.GetUser(c, &user) {
		for _, v := range user.InstallationIds {
			client := github.NewClient(&http.Client{
				Transport: ghinstallation.NewFromAppsTransport(lib.Itr, v),
				Timeout:   time.Second * 30,
			})
			installRepos, _, err := client.Apps.ListRepos(context.TODO(), &github.ListOptions{})
			if err != nil {
				log.Println(err)
				c.Redirect(http.StatusFound, "/")
				return
			}
			repos[v] = installRepos
		}
		if len(repos) == 0 {
			c.Redirect(http.StatusFound, lib.AppInstallUrl)
			return
		}
		c.HTML(http.StatusOK, "addRepoGithub", repos)
	} else {
		c.Redirect(http.StatusFound, "/login")
	}
}
