package lib

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation"
)

var Itr *ghinstallation.AppsTransport
var AppID int64

func StartGithubClient() {
	var err error
	Itr, err = ghinstallation.NewAppsTransport(http.DefaultTransport, AppID, []byte(os.Getenv("GITHUB_KEY")))
	if err != nil {
		log.Fatalf("faild to create app transport: %v\n", err)
	}
	AppID, err = strconv.ParseInt(os.Getenv("GITHUB_APP_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
}
