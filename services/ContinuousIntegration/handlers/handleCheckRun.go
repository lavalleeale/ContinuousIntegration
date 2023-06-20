package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/go-github/github"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/lib"
)

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
	for index, v := range buildData.Containers {
		buildData.Containers[index].Steps = append([]string{
			fmt.Sprintf("git checkout %s", *event.CheckRun.HeadSHA),
		}, v.Steps...)
	}
	if err != nil {
		// This should never fail since the webhook is directly from github and we know that our installation must be valid, and the panic will not be facing users
		panic(err)
	}
	build, err := lib.StartBuild(repo, buildData, []string{
		"x-access-token", token,
	})
	statuses := map[uint]string{}
	left := len(build.Containers)
	var detailsUrl string
	if os.Getenv("APP_ENV") == "production" {
		detailsUrl = fmt.Sprintf("https://%s/build/%d", host, build.ID)
	} else {
		detailsUrl = fmt.Sprintf("http://%s/build/%d", host, build.ID)
	}
	if err != nil {
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID, "ci.json not valid", err.Error(), completed, nil, &failure)
		return
	} else {
		for _, v := range build.Containers {
			statuses[v.Id] = fmt.Sprintf("\n|%s|⏳|", v.Name)
		}
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID,
			"Build Progress", generateMarkdown(statuses), inProgress, &detailsUrl, nil)
	}
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
		idStr := strings.Split(msg.Channel, ".")[3]
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			panic(err)
		}
		if msg.Payload == "0" {
			old := statuses[uint(id)]
			statuses[uint(id)] = old[:len(old)-4] + "✅|"
		} else {
			old := statuses[uint(id)]
			statuses[uint(id)] = old[:len(old)-4] + "❌|"
		}
		if left == 0 {
			break
		}
		updateCheckRun(client, *event.Repo.Owner.Login,
			*event.Repo.Name, *event.CheckRun.ID,
			"Build Progress", generateMarkdown(statuses), inProgress, &detailsUrl, nil)
	}

	msg := <-buildCh

	updateCheckRun(client, *event.Repo.Owner.Login,
		*event.Repo.Name, *event.CheckRun.ID,
		"Build Progress", generateMarkdown(statuses), inProgress, &detailsUrl, &msg.Payload)
}
