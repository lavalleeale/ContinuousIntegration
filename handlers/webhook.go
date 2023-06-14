package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/lib"
	"gorm.io/gorm"
)

func HandleWebhook(c *gin.Context) {
	payload, err := github.ValidatePayload(c.Request, []byte(os.Getenv("WEBHOOK_SECRET")))
	if err != nil {
		log.Println(err)
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(c.Request), payload)
	if err != nil {
		log.Println(err)
		return
	}
	switch event := event.(type) {
	case *github.CheckSuiteEvent:
		var repo db.Repo
		err = db.Db.First(&repo, "github_repo_id = ?", &event.Repo.ID).Error
		if err != nil {
			return
		}
		client := github.NewClient(&http.Client{
			Transport: ghinstallation.NewFromAppsTransport(lib.Itr, *event.Installation.ID),
			Timeout:   time.Second * 30,
		})
		if *event.Action == "requested" || *event.Action == "rerequested" {
			_, _, err := client.Checks.CreateCheckRun(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, github.CreateCheckRunOptions{Name: "Test", HeadSHA: *event.CheckSuite.HeadSHA})
			if err != nil {
				panic(err)
			}
		}
	case *github.CheckRunEvent:
		if *event.CheckRun.App.ID == lib.AppID {
			var repo db.Repo
			err = db.Db.First(&repo, "github_repo_id = ?", &event.Repo.ID).Error
			if err != nil {
				log.Println(err)
				return
			}
			transport := ghinstallation.NewFromAppsTransport(lib.Itr, *event.Installation.ID)
			client := github.NewClient(&http.Client{
				Transport: transport,
				Timeout:   time.Second * 30,
			})
			if *event.Action == "rerequested" {
				_, _, err := client.Checks.CreateCheckRun(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, github.CreateCheckRunOptions{Name: "Test", HeadSHA: *event.CheckRun.HeadSHA})
				if err != nil {
					panic(err)
				}
			} else if *event.Action == "created" {
				content, err := client.Repositories.DownloadContents(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, "/ci.json", &github.RepositoryContentGetOptions{Ref: *event.CheckRun.HeadSHA})
				if err != nil {
					panic(err)
				}
				json, err := io.ReadAll(content)
				if err != nil {
					panic(err)
				}
				gitConfig := ""
				var buildData = lib.BuildData{GitConfig: &gitConfig}
				err = binding.JSON.BindBody(json, &buildData.Containers)
				if err != nil {
					log.Println(err)
					return
				}
				for index, v := range buildData.Containers {
					buildData.Containers[index].Steps = append([]string{fmt.Sprintf("git checkout %s", *event.CheckRun.HeadSHA)}, v.Steps...)
				}
				token, err := transport.Token(context.TODO())
				if err != nil {
					log.Println(err)
					return
				}
				build, err := lib.StartBuild(repo, buildData, []string{"x-access-token", token}, func(id uint) {
					conclusion := "success"
					externalID := strconv.FormatInt(int64(id), 10)
					var detailsUrl string
					if os.Getenv("APP_ENV") == "production" {
						detailsUrl = fmt.Sprintf("https://%s/build/%d", c.Request.Host, id)
					} else {
						detailsUrl = fmt.Sprintf("http://%s/build/%d", c.Request.Host, id)
					}
					client.Checks.UpdateCheckRun(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, *event.CheckRun.ID, github.UpdateCheckRunOptions{Conclusion: &conclusion, ExternalID: &externalID, DetailsURL: &detailsUrl, Name: "Test"})
				})
				if err != nil {
					conclusion := "failure"
					client.Checks.UpdateCheckRun(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, *event.CheckRun.ID, github.UpdateCheckRunOptions{Conclusion: &conclusion})
				} else {
					status := "in_progress"
					externalID := strconv.FormatInt(int64(build.ID), 10)
					var detailsUrl string
					if os.Getenv("APP_ENV") == "production" {
						detailsUrl = fmt.Sprintf("https://%s/build/%d", c.Request.Host, build.ID)
					} else {
						detailsUrl = fmt.Sprintf("http://%s/build/%d", c.Request.Host, build.ID)
					}
					client.Checks.UpdateCheckRun(context.TODO(), *event.Repo.Owner.Login, *event.Repo.Name, *event.CheckRun.ID, github.UpdateCheckRunOptions{Status: &status, ExternalID: &externalID, DetailsURL: &detailsUrl, Name: "Test"})
				}
			}
		}
	case *github.InstallationEvent:
		if *event.Action == "deleted" {
			db.Db.Model(db.User{}).Where("1=1").Update("installation_ids", gorm.Expr("ARRAY_REMOVE(installation_ids, ?)", *event.Installation.ID))
		}
	default:
		log.Printf("Unknown request %s\n", c.Request.Header["X-Github-Event"])
	}
	c.String(200, "finished")
}
