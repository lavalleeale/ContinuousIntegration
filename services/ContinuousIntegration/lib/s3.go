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
	ctx := context.Background()
	minioClient, err := minio.New(os.Getenv("S3_URL"), &minio.Options{
		Creds: credentials.NewStaticV4(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
	})
	if err != nil {
		log.Fatalln(err)
	}
	MinioClient = minioClient
	err = MinioClient.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := MinioClient.BucketExists(ctx, BucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", BucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", BucketName)
	}
}
