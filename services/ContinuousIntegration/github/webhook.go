package github

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
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
				token, err := transport.Token(context.Background())
				if err != nil {
					// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
					panic(err)
				}
				HandleCheckRun(client, event, token, c.Request.Host)
			}
		}
	case *github.InstallationEvent:
		if *event.Action == "deleted" {
			// TODO: Remove from builds
			db.Db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(db.User{}).Update("installation_ids",
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
