package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/auth"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/github"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/handlers"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/ws"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	if os.Getenv("APP_ENV") == "PRODUCTION" {
		gin.SetMode(gin.ReleaseMode)
	}

	err = db.Open()
	if err != nil {
		log.Fatal("Failed to Open DB")
	}

	lib.InitTemplates()

	lib.StartDockerClient()

	lib.StartGithubClient()

	lib.StartRedisClient()

	lib.StartS3Client()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "addUser":
			log.Println(auth.Login(os.Args[2], os.Args[3], true))
		case "healthCheck":
			pages := GetTemplate().Templates()
			log.Printf("Loaded %d templates", len(pages))
			var repoCount int64
			db.Db.Model(db.Repo{}).Count(&repoCount)
			log.Printf("Repo Count: %d", repoCount)
			dockerResponse, err := lib.DockerCli.Ping(context.Background())
			if err != nil {
				log.Println("Docker Ping Failed")
			} else {
				log.Printf("Connected to docker version: %s\n", dockerResponse.APIVersion)
			}
			if lib.AppInstallUrl != "" {
				log.Printf("Github App Install URL: %s\n", lib.AppInstallUrl)
			} else {
				log.Println("Github App Install URL not set")
			}
			redisResponse := lib.Rdb.Ping(context.Background())
			if redisResponse.Err() != nil {
				log.Println("Redis Ping Failed")
			} else {
				log.Printf("Got redis ping response: %s\n", redisResponse.Val())
			}
			s3Response, err := lib.MinioClient.ListBuckets(context.Background())
			if err != nil {
				log.Println("S3 Ping Failed")
			} else {
				log.Println("S3 Buckets:")
				for _, bucket := range s3Response {
					log.Printf("Bucket: %s\n", bucket.Name)
				}
			}
		default:
			log.Println("Unknown Command")
		}
		return
	}

	serverRoot, err := fs.Sub(assetsFS, "assets/output")
	if err != nil {
		log.Fatal(err)
	}

	r := gin.New()
	r.Use(
		gin.Recovery(),
	)

	r.Use(lib.Session)

	r.SetHTMLTemplate(GetTemplate())

	r.POST("/github", github.HandleWebhook)

	r.GET("/callback", github.GithubCallback)

	r.GET("/addRepoGithub", github.AddRepoGithhubPage)

	r.POST("/addRepoGithub", github.AddRepoGithub)

	r.StaticFS("assets", http.FS(serverRoot))

	r.GET("/", handlers.IndexPage)

	r.GET("/repo/:repoId", handlers.RepoPage)

	r.GET("/build/:buildId", handlers.BuildPage)

	r.GET("/build/:buildId/container/:containerName", handlers.ContainerPage)

	r.GET("/login", handlers.LoginPage)
	r.POST("/login", handlers.Login)

	r.POST("/addRepo", handlers.AddRepo)

	r.POST("/repo/:repoId/build", handlers.StartBuild)

	r.GET("/build/:buildId/containerStatus", ws.HandleBuildWs)
	r.GET("/build/:buildId/container/:containerName/log", ws.HandleContainerWs)

	r.GET("/file/:fileId", handlers.DownloadFile)

	r.POST("/deleteRepo", handlers.DeleteRepo)

	r.GET("/unknownProxy", func(ctx *gin.Context) {
		ctx.String(http.StatusNotFound, "Unknown Proxy")
	})

	r.POST("/build/:buildId/container/:containerName/stop", handlers.StopContainer)

	r.GET("/invite", handlers.InvitePage)
	r.POST("/send-invite", handlers.InviteUser)
	r.GET("/acceptInvite", handlers.AcceptInvitePage)
	r.POST("/acceptInvite", handlers.AcceptInvite)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-done
	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}
