package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
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
			_, _, err := client.Checks.CreateCheckRun(context.TODO(),
				*event.Repo.Owner.Login, *event.Repo.Name, github.CreateCheckRunOptions{
					Name: "Test", HeadSHA: *event.CheckSuite.HeadSHA,
				})
			if err != nil {
				// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
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
				_, _, err := client.Checks.CreateCheckRun(context.TODO(),
					*event.Repo.Owner.Login, *event.Repo.Name, github.CreateCheckRunOptions{
						Name: "Test", HeadSHA: *event.CheckRun.HeadSHA,
					})
				if err != nil {
					// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
					panic(err)
				}
			} else if *event.Action == "created" {
				content, err := client.Repositories.DownloadContents(context.TODO(),
					*event.Repo.Owner.Login, *event.Repo.Name, "/ci.json", &github.RepositoryContentGetOptions{
						Ref: *event.CheckRun.HeadSHA,
					})
				if err != nil {
					if strings.Contains(err.Error(), "No file named") {
						// File not found
						conclusion := "failure"
						title := "ci.json not found!"
						summary := "ci.json file not found in root of repository"
						client.Checks.UpdateCheckRun(context.TODO(),
							*event.Repo.Owner.Login, *event.Repo.Name,
							*event.CheckRun.ID, github.UpdateCheckRunOptions{
								Conclusion: &conclusion, Name: "Test",
								Output: &github.CheckRunOutput{Title: &title, Summary: &summary},
							})
						return
					} else {
						// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
						panic(err)
					}
				}
				json, err := io.ReadAll(content)
				if err != nil {
					// Should never fail since file was downloaded successfully
					panic(err)
				}
				gitConfig := ""
				buildData := lib.BuildData{GitConfig: &gitConfig}
				err = binding.JSON.BindBody(json, &buildData.Containers)
				if err != nil {
					// Invalid JSON
					conclusion := "failure"
					title := "ci.json not valid"
					summary := err.Error()
					client.Checks.UpdateCheckRun(context.TODO(),
						*event.Repo.Owner.Login, *event.Repo.Name,
						*event.CheckRun.ID, github.UpdateCheckRunOptions{
							Conclusion: &conclusion, Name: "Test",
							Output: &github.CheckRunOutput{Title: &title, Summary: &summary},
						})
					return
				}
				for index, v := range buildData.Containers {
					buildData.Containers[index].Steps = append([]string{
						fmt.Sprintf("git checkout %s", *event.CheckRun.HeadSHA),
					}, v.Steps...)
				}
				token, err := transport.Token(context.TODO())
				if err != nil {
					// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
					panic(err)
				}
				build, err := lib.StartBuild(repo, buildData, []string{
					"x-access-token", token,
				}, func(id uint, failed bool) {
					var conclusion string
					if failed {
						db.Db.Model(db.Build{ID: id}).Update("status", "failed")
						conclusion = "failure"
					} else {
						db.Db.Model(db.Build{ID: id}).Update("status", "succeeded")
						conclusion = "success"
					}
					externalID := strconv.FormatInt(int64(id), 10)
					var detailsUrl string
					if os.Getenv("APP_ENV") == "production" {
						detailsUrl = fmt.Sprintf("https://%s/build/%d", c.Request.Host, id)
					} else {
						detailsUrl = fmt.Sprintf("http://%s/build/%d", c.Request.Host, id)
					}
					client.Checks.UpdateCheckRun(context.TODO(),
						*event.Repo.Owner.Login, *event.Repo.Name,
						*event.CheckRun.ID, github.UpdateCheckRunOptions{
							Conclusion: &conclusion, ExternalID: &externalID, DetailsURL: &detailsUrl, Name: "Test",
						})
				})
				if err != nil {
					conclusion := "failure"
					title := "ci.json not valid"
					summary := err.Error()
					client.Checks.UpdateCheckRun(context.TODO(),
						*event.Repo.Owner.Login, *event.Repo.Name,
						*event.CheckRun.ID, github.UpdateCheckRunOptions{
							Conclusion: &conclusion, Name: "Test",
							Output: &github.CheckRunOutput{Title: &title, Summary: &summary},
						})
				} else {
					status := "in_progress"
					externalID := strconv.FormatInt(int64(build.ID), 10)
					var detailsUrl string
					if os.Getenv("APP_ENV") == "production" {
						detailsUrl = fmt.Sprintf("https://%s/build/%d", c.Request.Host, build.ID)
					} else {
						detailsUrl = fmt.Sprintf("http://%s/build/%d", c.Request.Host, build.ID)
					}
					client.Checks.UpdateCheckRun(context.TODO(),
						*event.Repo.Owner.Login, *event.Repo.Name,
						*event.CheckRun.ID, github.UpdateCheckRunOptions{
							Status: &status, ExternalID: &externalID, DetailsURL: &detailsUrl, Name: "Test",
						})
				}
			}
		}
	case *github.InstallationEvent:
		if *event.Action == "deleted" {
			// TODO: Remove from builds
			// Hack needed because GORM will not update with where clause but is acceptable since ARRAY_REMOVE will only remove given ID
			db.Db.Model(db.User{}).Where("1 = 1").Update("installation_ids",
				gorm.Expr("ARRAY_REMOVE(installation_ids, ?)", *event.Installation.ID))
			db.Db.Model(db.Repo{}).Select("installation_id", "github_repo_id").Where(
				"installation_id = ?", *event.Installation.ID).Updates(db.Repo{
				InstallationId: nil, GithubRepoId: nil,
			})
		}
	default:
		log.Printf("Unknown request %s\n", c.Request.Header["X-Github-Event"])
	}
	c.String(200, "finished")
}
