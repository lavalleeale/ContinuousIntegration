package handlers

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
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

	if lib.GetUser(c, &user) {
		repoId, err := strconv.Atoi(c.Param("repoId"))
		if err != nil {
			c.Redirect(http.StatusFound, "/")
		}

		repo := db.Repo{ID: uint(repoId)}

		tx := db.Db.Preload("Builds").First(&repo)

		if tx.Error == nil && repo.OrganizationID == user.OrganizationID {
			c.HTML(http.StatusOK, "repo", repo)
			return
		}
	}
	c.Redirect(http.StatusFound, "/")
}

func BuildPage(c *gin.Context) {
	var user db.User

	if lib.GetUser(c, &user) {

		buildId, err := strconv.Atoi(c.Param("buildId"))
		if err == nil {

			build := db.Build{ID: uint(buildId), Repo: db.Repo{OrganizationID: user.OrganizationID}}
			tx := db.Db.Preload("Repo").Preload("Containers").Preload(
				"Containers.UploadedFiles", func(db *gorm.DB) *gorm.DB {
					return db.Select("id", "path", "container_id")
				}).First(&build)

			if err == nil && build.Repo.OrganizationID == user.OrganizationID {
				if tx.Error == nil || !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
					buildPageData := BuildPageData{Build: build, PersistHost: os.Getenv("PERSIST_HOST")}
					if os.Getenv("APP_ENV") == "production" {
						buildPageData.PersistScheme = "https"
					} else {
						buildPageData.PersistScheme = "http"
					}
					c.HTML(http.StatusOK, "build", buildPageData)
					return
				}
			}
		}
	}
	c.Redirect(http.StatusFound, "/")
}

func ContainerPage(c *gin.Context) {
	var user db.User

	if lib.GetUser(c, &user) {
		containerId, err := strconv.Atoi(c.Param("containerId"))
		if err != nil {
			c.Redirect(http.StatusFound, "/")
		}

		container := db.Container{Id: uint(containerId)}

		db.Db.Preload("Build.Repo").First(&container)

		if container.Build.Repo.OrganizationID == user.OrganizationID {
			c.HTML(http.StatusOK, "container", container)
			return
		}
	}
	c.Redirect(http.StatusFound, "/")
}

type BuildPageData struct {
	Build         db.Build
	PersistScheme string
	PersistHost   string
}
