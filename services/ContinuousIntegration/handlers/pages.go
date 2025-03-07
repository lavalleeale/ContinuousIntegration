package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	sessionseal "github.com/lavalleeale/SessionSeal"
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

			build := db.Build{ID: uint(buildId)}
			tx := db.Db.Preload("Repo").Preload("Containers.EdgesFrom").Preload(
				"Containers.FilesUploaded", func(db *gorm.DB) *gorm.DB {
					return db.Select("id", "path", "from_name", "build_id")
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
		buildId, err := strconv.Atoi(c.Param("buildId"))
		if err != nil {
			c.Redirect(http.StatusFound, "/")
		}

		container := db.Container{Name: c.Param("containerName"), BuildID: uint(buildId)}

		db.Db.Preload("Build.Repo").First(&container)

		if container.Build.Repo.OrganizationID == user.OrganizationID {
			c.HTML(http.StatusOK, "container", container)
			return
		}
	}
	c.Redirect(http.StatusFound, "/")
}

func InvitePage(c *gin.Context) {
	var user db.User

	if lib.GetUser(c, &user) {
		c.HTML(http.StatusOK, "sendInvite", gin.H{})
		return
	}
	c.Redirect(http.StatusFound, "/")
}

func AcceptInvitePage(c *gin.Context) {
	// Retrieve sealed invite data from query parameter
	sealedData := c.Query("data")
	if sealedData == "" {
		c.String(http.StatusBadRequest, "Missing invite data")
		return
	}

	// Unseal and decode the invite data using JWT_SECRET
	marshalledInvite, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), sealedData)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid invite token")
		return
	}

	var invite db.OrganizationInvite
	if err := json.Unmarshal(marshalledInvite, &invite); err != nil {
		c.String(http.StatusBadRequest, "Invalid invite data")
		return
	}

	// Render the invite acceptance page
	c.HTML(http.StatusOK, "acceptInvite", gin.H{
		"data":   sealedData,
		"invite": invite,
	})
}

type BuildPageData struct {
	Build         db.Build
	PersistScheme string
	PersistHost   string
}
