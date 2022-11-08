package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func IndexPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/login")
		return
	} else {
		db.Db.Preload(c, &user, "organization")
		db.Db.Preload(c, &user, "organization.repos")
		c.HTML(http.StatusOK, "dash", gin.H{"repos": user.Organization.Repos})
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

	var repo db.Repo

	db.Db.Find(c, &repo, rel.Eq("id", c.Param("repoId")))

	if repo.OrganizationID == user.OrganizationID {
		db.Db.Preload(c, &repo, "builds")
		c.HTML(http.StatusOK, "repo", repo)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}

func BuildPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	var build db.Build

	db.Db.Find(c, &build, rel.Eq("id", c.Param("buildId")))

	db.Db.Preload(c, &build, "repo")

	if build.Repo.OrganizationID == user.OrganizationID {
		db.Db.Preload(c, &build, "containers")
		c.HTML(http.StatusOK, "build", build)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}

func ContainerPage(c *gin.Context) {
	var user db.User

	lib.GetUser(c, &user)

	var container db.Container

	db.Db.Find(c, &container, rel.And(rel.Eq("id", c.Param("containerId")), rel.Eq("build_id", c.Param("buildId"))))

	db.Db.Preload(c, &container, "build")
	db.Db.Preload(c, &container, "build.repo")

	if container.Build.Repo.OrganizationID == user.OrganizationID {
		c.HTML(http.StatusOK, "container", container)
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}
