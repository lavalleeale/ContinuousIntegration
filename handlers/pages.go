package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
	"gorm.io/gorm"
)

func IndexPage(c *gin.Context) {
	var user db.User

	if !lib.GetUser(c, &user) {
		c.Redirect(http.StatusFound, "/login")
		return
	} else {
		organization := db.Organization{ID: user.OrganizationID}
		db.Db.Preload("Repos").First(&organization)
		c.HTML(http.StatusOK, "dash", gin.H{"repos": organization.Repos})
	}
}

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login", gin.H{
		"error": "",
	})
}

func RepoPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	repoId, err := strconv.Atoi(c.Param("repoId"))

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	repo := db.Repo{ID: uint(repoId), OrganizationID: user.OrganizationID}

	tx := db.Db.Preload("Builds").Where(&repo, "id", "OrganizationID").First(&repo)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		c.Redirect(http.StatusFound, "/")
	} else {
		c.HTML(http.StatusOK, "repo", repo)
	}
}

func BuildPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	buildId, err := strconv.Atoi(c.Param("buildId"))

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	build := db.Build{ID: uint(buildId), Repo: db.Repo{OrganizationID: user.OrganizationID}}

	tx := db.Db.Preload("Repo").Preload("Containers").Where(&build, "id", "repo.organizationID").First(&build)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		c.Redirect(http.StatusFound, "/")
	} else {
		c.HTML(http.StatusOK, "build", build)
	}
}

func ContainerPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	containerId, err := strconv.Atoi(c.Param("containerId"))

	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	container := db.Container{Id: uint(containerId), Build: db.Build{Repo: db.Repo{OrganizationID: user.OrganizationID}}}

	db.Db.Preload("Build.Repo").Where(&container, "id", "Build.Repo.OrganizationID").First(&container)

	if container.Build.Repo.OrganizationID == user.OrganizationID {
		c.HTML(http.StatusOK, "container", container)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
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
		c.HTML(http.StatusOK, "addRepoGithub", repos)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}
