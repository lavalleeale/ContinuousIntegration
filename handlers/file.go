package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
)

func DownloadFile(c *gin.Context) {
	user := db.User{}
	if !lib.GetUser(c, &user) {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	fileIdString := c.Param("fileId")
	fileId, err := uuid.Parse(fileIdString)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid File ID")
		return
	}
	file := db.UploadedFile{ID: fileId}
	tx := db.Db.Preload("Container.Build.Repo").First(&file)
	if tx.Error != nil {
		c.String(http.StatusBadRequest, "Invalid File ID")
		return
	}
	if file.Container.Build.Repo.OrganizationID != user.OrganizationID {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	c.Data(http.StatusOK, "application/x-tar", file.Bytes)
}
