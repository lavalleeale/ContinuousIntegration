package lib

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

var (
	Itr           *ghinstallation.AppsTransport
	AppID         int64
	AppInstallUrl string
)

func StartGithubClient() {
	var err error
	AppID, err = strconv.ParseInt(os.Getenv("GITHUB_APP_ID"), 10, 64)
	if err != nil {
		log.Println("Running without github client! This should only be done during tests!")
		return
	}
	Itr, err = ghinstallation.NewAppsTransport(http.DefaultTransport, AppID, []byte(os.Getenv("GITHUB_KEY")))
	if err != nil {
		log.Fatalf("faild to create app transport: %v\n", err)
	}
	client := github.NewClient(&http.Client{
		Transport: Itr,
		Timeout:   time.Second * 30,
	})
	app, _, err := client.Apps.Get(context.TODO(), "")
	if err != nil {
		panic(err)
	}
	AppInstallUrl = fmt.Sprintf("https://github.com/apps/%s/installations/new",
		strings.ReplaceAll((*app.Name), " ", "-"))
}
