package github

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

var (
	failure    = "failure"
	inProgress = "in_progress"
	completed  = "completed"
)

func updateCheckRun(
	client *github.Client,
	owner string,
	repo string,
	checkRunId int64,
	title string,
	summary string,
	status string,
	detailsUrl *string,
	conclusion *string,
) {
	client.Checks.UpdateCheckRun(context.TODO(),
		owner, repo,
		checkRunId, github.UpdateCheckRunOptions{
			Status:     &status,
			Name:       "Test",
			Output:     &github.CheckRunOutput{Title: &title, Summary: &summary},
			DetailsURL: detailsUrl,
			Conclusion: conclusion,
		})
}

func HandleCheckRun(client *github.Client, event *github.CheckRunEvent, token string, host string) {
	var repo db.Repo
	err := db.Db.First(&repo, "github_repo_id = ?", &event.Repo.ID).Error
	if err != nil {
		log.Println(err)
		return
	}
	content, err := client.Repositories.DownloadContents(context.TODO(),
		*event.Repo.Owner.Login, *event.Repo.Name, "ci.json", &github.RepositoryContentGetOptions{
			Ref: *event.CheckRun.HeadSHA,
		})
	if err != nil {
		if strings.Contains(err.Error(), "No file named") {
			updateCheckRun(client, *event.Repo.Owner.Login,
				*event.Repo.Name, *event.CheckRun.ID,
				"ci.json file not found", "ci.json file not found in root of repository", completed, nil, &failure)
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
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID, "ci.json not valid", err.Error(), completed, nil, &failure)
		return
	}
	for _, v := range buildData.Containers {
		for _, step := range v.Steps {
			if step.Type == "clone" {
				step.Sha = *event.CheckRun.HeadSHA
			}
		}
	}
	if err != nil {
		// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
		panic(err)
	}
	build, edges, err := lib.StartBuild(repo, buildData, []string{
		"x-access-token", token,
	})
	if err != nil {
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID, "ci.json not valid", err.Error(), completed, nil, &failure)
		return
	}
	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("```mermaid\ngraph LR;\n")
	for _, v := range build.Containers {
		markdownBuilder.WriteString(fmt.Sprintf("%s(\"%s ⏳\");\n", v.Name, v.Name))
	}
	for _, v := range edges {
		markdownBuilder.WriteString(fmt.Sprintf("%s-->%s;\n", v.FromName, v.ToName))
	}
	left := len(build.Containers)
	var detailsUrl string
	if os.Getenv("APP_ENV") == "production" {
		detailsUrl = fmt.Sprintf("https://%s/build/%d", host, build.ID)
	} else {
		detailsUrl = fmt.Sprintf("http://%s/build/%d", host, build.ID)
	}
	updateCheckRun(client, *event.Repo.Owner.Login,
		*event.Repo.Name, *event.CheckRun.ID,
		"Build Progress", markdownBuilder.String()+"```", inProgress, &detailsUrl, nil)
	containersPubsub := lib.Rdb.PSubscribe(context.TODO(),
		fmt.Sprintf("build.%d.*.die", build.ID))
	defer containersPubsub.Close()
	containersCh := containersPubsub.Channel()

	// Open channel early to make sure we get event
	buildPubsub := lib.Rdb.Subscribe(context.TODO(), fmt.Sprintf("build.%d", build.ID))
	defer buildPubsub.Close()
	buildCh := buildPubsub.Channel()

	for msg := range containersCh {
		left--
		name := strings.Split(msg.Channel, ".")[3]
		if msg.Payload == "0" {
			markdownBuilder.WriteString(fmt.Sprintf("%s(\"%s %s\");\n", name, name, "✅"))
		} else {
			markdownBuilder.WriteString(fmt.Sprintf("%s(\"%s %s\");\n", name, name, "❌"))
		}
		if left == 0 {
			break
		}
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID,
			"Build Progress", markdownBuilder.String()+"```", inProgress, &detailsUrl, nil)
	}

	msg := <-buildCh

	updateCheckRun(client, *event.Repo.Owner.Login,
		*event.Repo.Name, *event.CheckRun.ID,
		"Build Progress", markdownBuilder.String()+"```", completed, &detailsUrl, &msg.Payload)
}
