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

	if len(os.Args) > 1 && os.Args[1] == "addUser" {
		log.Println(auth.Login(os.Args[2], os.Args[3], true))
		return
	}

	lib.StartDockerClient()

	lib.StartGithubClient()

	lib.StartRedisClient()

	serverRoot, err := fs.Sub(assetsFS, "assets/output")
	if err != nil {
		log.Fatal(err)
	}

	r := gin.New()

	r.Use(gin.Recovery(), lib.Session)

	r.SetHTMLTemplate(GetTemplate())

	r.POST("/github", github.HandleWebhook)

	r.GET("/callback", github.GithubCallback)

	r.GET("/addRepoGithub", github.AddRepoGithhubPage)

	r.POST("/addRepoGithub", github.AddRepoGithub)

	r.StaticFS("assets", http.FS(serverRoot))

	r.GET("/", handlers.IndexPage)

	r.GET("/repo/:repoId", handlers.RepoPage)

	r.GET("/build/:buildId", handlers.BuildPage)

	r.GET("/build/:buildId/container/:containerId", handlers.ContainerPage)

	r.GET("/login", handlers.LoginPage)
	r.POST("/login", handlers.Login)

	r.POST("/addRepo", handlers.AddRepo)

	r.POST("/repo/:repoId/build", handlers.StartBuild)

	r.GET("/build/:buildId/containerStatus", ws.HandleBuildWs)
	r.GET("/build/:buildId/container/:containerId/log", ws.HandleContainerWs)

	r.GET("/file/:fileId", handlers.DownloadFile)

	r.POST("/deleteRepo", handlers.DeleteRepo)

	r.GET("/unknownProxy", func(ctx *gin.Context) {
		ctx.String(http.StatusNotFound, "Unknown Proxy")
	})

	r.POST("/build/:buildId/container/:containerId/stop", handlers.StopContainer)

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
