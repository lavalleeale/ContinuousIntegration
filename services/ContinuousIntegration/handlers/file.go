package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	"github.com/minio/minio-go/v7"
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
	tx := db.Db.Preload("Build.Repo").First(&file)
	if tx.Error != nil {
		c.String(http.StatusBadRequest, "Invalid File ID")
		return
	}
	if file.Build.Repo.OrganizationID != user.OrganizationID {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	object, err := lib.MinioClient.GetObject(context.TODO(), lib.BucketName, file.ID.String(), minio.GetObjectOptions{})
	if err != nil {
		c.String(http.StatusNotFound, "File Not Found")
		return
	}

	bytes, err := io.ReadAll(object)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error Reading File")
		return
	}

	c.Data(http.StatusOK, "application/x-tar", bytes)
}
