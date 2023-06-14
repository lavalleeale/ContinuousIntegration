package lib

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/heimdalr/dag"
	"github.com/lavalleeale/ContinuousIntegration/db"
)

type BuildData struct {
	GitConfig  *string `json:"gitConfig"`
	Containers []struct {
		ID          string               `json:"id"`
		Steps       []string             `json:"steps"`
		Image       string               `json:"image"`
		Environment *[]map[string]string `json:"environment,omitempty"`
		Needs       *[]string            `json:"needs,omitempty"`
		NeededFiles *[]string            `json:"neededFiles,omitempty"`
		Uploads     *[]string            `json:"uploads,omitempty"`
		Service     *struct {
			Steps       *[]string            `json:"steps,omitempty"`
			Environment *[]map[string]string `json:"environment,omitempty"`
			Image       string               `json:"image"`
			Healthcheck string               `json:"healthcheck"`
		} `json:"service,omitempty"`
	} `json:"containers"`
}

func StartBuild(repo db.Repo, buildData BuildData, auth []string, callback func(uint)) (db.Build, error) {
	var build = db.Build{RepoID: repo.ID, Containers: make([]db.Container, len(buildData.Containers)), GitConfig: ""}

	if buildData.GitConfig != nil {
		build.GitConfig = *buildData.GitConfig
	}

	d := dag.NewDAG()

	for index, container := range buildData.Containers {
		savedContainer := db.Container{
			Name:    container.ID,
			Command: strings.Join(container.Steps, " && "),
			Image:   container.Image,
		}
		if container.Environment != nil {
			var environment = make([]string, 0)
			for _, item := range *container.Environment {
				for k, v := range item {
					environment = append(environment, fmt.Sprintf("%s=%s", k, v))
				}
			}
			environmentString := strings.Join(environment, ",")
			savedContainer.Environment = &environmentString
		}
		if container.Service != nil {
			savedContainer.ServiceImage = &container.Service.Image
			savedContainer.ServiceHealthcheck = &container.Service.Healthcheck
			if container.Service.Steps != nil {
				command := strings.Join(*container.Service.Steps, " && ")
				savedContainer.ServiceCommand = &command
			}
			if container.Service.Environment != nil {
				var environment = make([]string, 0)
				for _, item := range *container.Service.Environment {
					for k, v := range item {
						environment = append(environment, fmt.Sprintf("%s=%s", k, v))
					}
				}
				environmentString := strings.Join(environment, ",")
				savedContainer.ServiceEnvironment = &environmentString
			}
		}
		if container.Uploads != nil {
			uploadedFiles := make([]db.UploadedFile, len(*container.Uploads))
			for index, uploadedFile := range *container.Uploads {
				uploadedFiles[index] = db.UploadedFile{Path: uploadedFile}
			}
			savedContainer.UploadedFiles = uploadedFiles
		}
		build.Containers[index] = savedContainer
		d.AddVertex(&build.Containers[index])
	}

	for index, container := range buildData.Containers {
		if container.Needs != nil {
			for _, need := range *container.Needs {
				err := d.AddEdge(need, container.ID)
				if err != nil {
					//TODO: deal with
					panic(err)
				}
			}
			if container.NeededFiles != nil {
				for _, neededFile := range *container.NeededFiles {
					ancestors, err := d.GetAncestors(container.ID)
					if err != nil {
						panic(err)
					}
					found := false
					for k := range ancestors {
						split := strings.Split(neededFile, ":")
						if k == split[0] {
							build.Containers[index].NeededFiles = append(build.Containers[index].NeededFiles, db.NeededFile{From: k, FromPath: split[1]})
							found = true
							break
						}
					}
					if !found {
						//TODO: deal with
						panic(found)
					}
				}
			}
		} else if container.NeededFiles != nil {
			//TODO: deal with
			panic("Needs files but does not have ancestors")
		}
	}

	err := db.Db.Create(&build).Error

	if err != nil {
		return db.Build{}, err
	}

	edges := make([]db.ContainerGraphEdge, 0)
	for _, node := range d.GetRoots() {
		treeWalk(d, *node.(*db.Container), &edges)
	}

	if len(edges) != 0 {
		tx := db.Db.Create(&edges)
		if tx.Error != nil {
			panic(tx.Error)
		}
	}
	go (func() {
		var wg sync.WaitGroup
		wg.Add(1)
		for _, container := range d.GetRoots() {

			if len(auth) != 0 {
				repoUrl, err := url.Parse(repo.Url)
				if err != nil {
					panic(err)
				}
				repoUrl.User = url.UserPassword(auth[0], auth[1])
				go BuildContainer(repoUrl.String(), build.ID, *container.(*db.Container), &wg)
			} else {
				go BuildContainer(repo.Url, build.ID, *container.(*db.Container), &wg)
			}
		}
		wg.Wait()
		if callback != nil {
			callback(build.ID)
		}
	})()
	return build, nil
}

func treeWalk(d *dag.DAG, startNode db.Container, a *[]db.ContainerGraphEdge) {
	children, err := d.GetChildren(startNode.Name)
	if err != nil {
		panic(err)
	}
	for _, childNode := range children {
		*a = append(
			*a,
			db.ContainerGraphEdge{
				FromID: uint(startNode.Id),
				ToID:   uint(childNode.(*db.Container).Id),
			},
		)
		treeWalk(d, *childNode.(*db.Container), a)
	}
}
