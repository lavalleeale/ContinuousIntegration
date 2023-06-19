package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/handlers"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/ws"
)

//go:embed assets/output/*
var assetsFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

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

	lib.StartDockerClient()

	lib.StartGithubClient()

	lib.StartRedisClient()

	serverRoot, err := fs.Sub(assetsFS, "assets/output")
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	r.Use(lib.Session)

	r.POST("/github", handlers.HandleWebhook)

	r.GET("/callback", handlers.GithubCallback)

	r.GET("/addRepoGithub", handlers.AddRepoGithhubPage)

	r.POST("/addRepoGithub", handlers.AddRepoGithub)

	funcMap := template.FuncMap{
		"Deref": func(i *int) int { return *i },
	}

	r.SetHTMLTemplate(template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/**/*")))

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

	r.Run()
}
