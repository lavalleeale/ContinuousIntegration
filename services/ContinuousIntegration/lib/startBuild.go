package lib

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"

	"github.com/heimdalr/dag"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
)

type BuildData struct {
	GitConfig  *string `json:"gitConfig"`
	Containers []struct {
		ID          string             `json:"id"`
		Steps       []TemplateData     `json:"steps"`
		Image       string             `json:"image"`
		Environment *map[string]string `json:"environment,omitempty"`
		Needs       *[]string          `json:"needs,omitempty"`
		NeededFiles *[]string          `json:"neededFiles,omitempty"`
		Uploads     *[]string          `json:"uploads,omitempty"`
		Persist     *bool              `json:"persist,omitempty"`
		Services    *map[string]struct {
			Steps       *[]string          `json:"steps,omitempty"`
			Environment *map[string]string `json:"environment,omitempty"`
			Image       string             `json:"image"`
			Healthcheck string             `json:"healthcheck"`
		} `json:"services,omitempty"`
	} `json:"containers"`
}

func StartBuild(repo db.Repo, buildData BuildData, auth []string) (db.Build, []db.ContainerGraphEdge, error) {
	build := db.Build{RepoID: repo.ID, Containers: make([]db.Container,
		len(buildData.Containers)), GitConfig: ""}

	if buildData.GitConfig != nil {
		build.GitConfig = *buildData.GitConfig
	}

	d := dag.NewDAG()

	for index, container := range buildData.Containers {
		if len(container.Steps) == 0 {
			return db.Build{}, nil, fmt.Errorf("container %s has no steps", container.ID)
		}
		var persist *string
		if container.Persist != nil && *container.Persist {
			persistString := fmt.Sprintf("%x", rand.Uint64())
			persist = &persistString
		}
		steps := make([]string, len(container.Steps))
		for index, step := range container.Steps {
			var err error
			steps[index], err = step.GetString(repo)
			if err != nil {
				return db.Build{}, nil, err
			}
		}
		savedContainer := db.Container{
			Name:    container.ID,
			Command: strings.Join(steps, " && "),
			Image:   container.Image,
			Persist: persist,
		}
		if container.Environment != nil {
			environment := make([]string, 0)
			for k, v := range *container.Environment {
				environment = append(environment, fmt.Sprintf("%s=%s", k, v))
			}
			environmentString := strings.Join(environment, ",")
			savedContainer.Environment = &environmentString
		}
		if container.Services != nil {
			for name, data := range *container.Services {
				newContainer := db.ServiceContainer{Name: name, Image: data.Image, Healthcheck: data.Healthcheck}
				if data.Steps != nil {
					command := strings.Join(*data.Steps, " && ")
					newContainer.Command = &command
				}
				if data.Environment != nil {
					environment := make([]string, 0)
					for k, v := range *data.Environment {
						environment = append(environment, fmt.Sprintf("%s=%s", k, v))
					}
					environmentString := strings.Join(environment, ",")
					newContainer.Environment = &environmentString
				}
				savedContainer.ServiceContainers = append(
					savedContainer.ServiceContainers, newContainer)
			}
		}
		if container.Uploads != nil {
			uploadedFiles := make([]db.UploadedFile, len(*container.Uploads))
			for index, uploadedFile := range *container.Uploads {
				uploadedFiles[index] = db.UploadedFile{Path: uploadedFile, FromName: container.ID}
			}
			savedContainer.FilesUploaded = uploadedFiles
		}
		build.Containers[index] = savedContainer
		d.AddVertex(&build.Containers[index])
	}

	for _, container := range buildData.Containers {
		if container.Needs != nil {
			for _, need := range *container.Needs {
				needed, err := d.GetVertex(need)
				if err != nil {
					return db.Build{}, nil, err
				}
				if (needed.(*db.Container)).Persist != nil {
					return db.Build{}, nil, fmt.Errorf("%s needs %s but %s is marked as persist", container.ID, need, need)
				}
				err = d.AddEdge(need, container.ID)
				if err != nil {
					return db.Build{}, nil, err
				}
			}
			if container.NeededFiles != nil {
				dbContainer, err := d.GetVertex(container.ID)
				if err != nil {
					panic(err)
				}
				for _, neededFile := range *container.NeededFiles {
					ancestors, err := d.GetAncestors(container.ID)
					if err != nil {
						// Getting ancestors should never fail even if slice is empty
						panic(err)
					}
					found := false
					split := strings.Split(neededFile, ":")
					for k := range ancestors {
						if k == split[0] {
							uploadFound := false
							ancestor := ancestors[k].(*db.Container)
							for index, uploadedFile := range ancestor.FilesUploaded {
								if uploadedFile.Path == split[1] {
									ancestor.FilesUploaded[index].To = append(
										ancestor.FilesUploaded[index].To, dbContainer.(*db.Container))
									uploadFound = true
									break
								}
							}
							if !uploadFound {
								return db.Build{}, nil, fmt.Errorf("%s needs file %s from %s however %s does not upload %s",
									container.ID, split[1], split[0], split[0], split[1])
							}
							found = true
							break
						}
					}
					if !found {
						return db.Build{}, nil, fmt.Errorf("%s needs file from %s however %s was not found in acestors", container.ID, split[0], split[0])
					}
				}
			}
		} else if container.NeededFiles != nil {
			return db.Build{}, nil, fmt.Errorf("%s needs files but does not have acestors", container.ID)
		}
	}

	err := db.Db.Create(&build).Error
	if err != nil {
		return db.Build{}, nil, fmt.Errorf("failed to create build")
	}

	edges := make([]db.ContainerGraphEdge, 0)
	for _, node := range d.GetRoots() {
		treeWalk(d, *node.(*db.Container), &edges)
	}

	if len(edges) != 0 {
		tx := db.Db.Create(&edges)
		if tx.Error != nil {
			return db.Build{}, nil, fmt.Errorf("failed to create build")
		}
	}

	repoUrl, err := url.Parse(repo.Url)
	if err != nil {
		return db.Build{}, nil, fmt.Errorf("unable to parse repo url %s", repo.Url)
	}
	go (func() {
		var wg sync.WaitGroup
		failed := false
		for _, container := range d.GetRoots() {
			wg.Add(1)
			if len(auth) != 0 {
				repoUrl.User = url.UserPassword(auth[0], auth[1])
				go BuildContainer(repoUrl.String(),
					*container.(*db.Container), repo.OrganizationID, &wg, &failed)
			} else {
				go BuildContainer(repo.Url, *container.(*db.Container), repo.OrganizationID, &wg, &failed)
			}
		}
		wg.Wait()
		if failed {
			db.Db.Model(&build).Update("status", "failure")
		} else {
			db.Db.Model(&build).Update("status", "success")
		}
		err = Rdb.Publish(context.TODO(), fmt.Sprintf("build.%d", build.ID), build.Status).Err()
		if err != nil {
			panic(err)
		}
	})()
	return build, edges, nil
}

func treeWalk(d *dag.DAG, startNode db.Container, a *[]db.ContainerGraphEdge) {
	children, err := d.GetChildren(startNode.Name)
	if err != nil {
		// Should never fail since tree was already made
		panic(err)
	}
	for _, childNode := range children {
		*a = append(
			*a,
			db.ContainerGraphEdge{
				FromName: startNode.Name,
				ToName:   childNode.(*db.Container).Name,
				BuildID:  startNode.BuildID,
			},
		)
		treeWalk(d, *childNode.(*db.Container), a)
	}
}
