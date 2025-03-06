package lib

import (
	"context"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	MinioClient *minio.Client
	BucketName  = "uploaded-files"
)

func StartS3Client() {
	var err error
	ctx := context.Background()
	MinioClient, err = minio.New(os.Getenv("S3_URL"), &minio.Options{
		Creds: credentials.NewStaticV4(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
	})
	if err != nil {
		log.Fatalln(err)
	}
	exists, err := MinioClient.BucketExists(ctx, BucketName)
	if err != nil {
		log.Fatalln(err)
	}
	if !exists {
		MinioClient.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{})
	}
}
