package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/lavalleeale/ContinuousIntegration/lib/db"
)

const (
	Clone = "git {{ .GitConfig }} clone {{ .Url }} /repo && cd /repo{{ if .Sha }} && git checkout {{ .Sha }} {{ end }}"
	// Clone       = "GIT_SSL_NO_VERIFY=1 git {{ .GitConfig }} clone {{ .Url }} /repo && cd /repo"
	Docker      = "(dockerd > dockerlog.txt 2>&1 &) && until docker version >/dev/null 2>&1; do sleep 1; done"
	Shell       = "{{ .Command }}"
	DockerBuild = `{{ $registry := (or .Registry "$REGISTRY") }}docker buildx create --driver docker-container --name mybuilder --use --bootstrap && \
docker cp /usr/local/share/ca-certificates/registry.crt buildx_buildkit_mybuilder0:/usr/local/share/ca-certificates/registry.crt &&
docker exec buildx_buildkit_mybuilder0 update-ca-certificates &&
docker login --username={{ or .Username "$DOCKER_USER" }} --password={{ or .Password "$DOCKER_PASS" }} {{ $registry }} &&
docker buildx build --tag {{ .Tag }} --cache-to=type=registry,ref={{ $registry }}/{{ .CacheTag }},mode=max --cache-from=type=registry,ref={{ $registry }}/{{ .CacheTag }} --load images/base`
)

type CloneTemplateData struct {
	GitConfig string `json:"gitConfig,omitempty"`
	Sha       string `json:"sha,omitempty"`
}

type DockerTemplateData struct{}

type ShellTemplateData struct {
	Command string `json:"command"`
}

type DockerBuildTemplateData struct {
	Registry string `json:"registry,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Tag      string `json:"tag"`
	CacheTag string `json:"cacheTag"`
}

type TypeSwitch struct {
	Type string `json:"type"`
}

type TemplateData struct {
	TypeSwitch
	*CloneTemplateData
	*DockerTemplateData
	*ShellTemplateData
	*DockerBuildTemplateData
}

func (templateData *TemplateData) GetString(repo db.Repo) (string, error) {
	var buf bytes.Buffer
	switch templateData.Type {
	case "clone":
		if err := CloneTemplate.Execute(&buf, struct {
			CloneTemplateData
			Url string
		}{
			CloneTemplateData: *templateData.CloneTemplateData,
			Url:               repo.Url,
		}); err != nil {
			return "", err
		}
	case "start-docker":
		if err := DockerTemplate.Execute(&buf, templateData.DockerTemplateData); err != nil {
			return "", err
		}
	case "shell":
		if err := ShellTemplate.Execute(&buf, templateData.ShellTemplateData); err != nil {
			return "", err
		}
	case "build-docker":
		if err := DockerBuildTemplate.Execute(&buf, templateData.DockerBuildTemplateData); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

var (
	CloneTemplate       *template.Template
	DockerTemplate      *template.Template
	ShellTemplate       *template.Template
	DockerBuildTemplate *template.Template
)

func InitTemplates() {
	CloneTemplate = template.Must(template.New("clone").Parse(Clone))
	DockerTemplate = template.Must(template.New("start-docker").Parse(Docker))
	ShellTemplate = template.Must(template.New("shell").Parse(Shell))
	DockerBuildTemplate = template.Must(template.New("build-docker").Parse(DockerBuild))
}

func (templateData *TemplateData) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &templateData.TypeSwitch); err != nil {
		return err
	}
	switch templateData.Type {
	case "clone":
		var cloneTemplateData CloneTemplateData
		if err := json.Unmarshal(data, &cloneTemplateData); err != nil {
			return err
		}
		templateData.CloneTemplateData = &cloneTemplateData
		return nil
	case "start-docker":
		var dockerTemplateData DockerTemplateData
		if err := json.Unmarshal(data, &dockerTemplateData); err != nil {
			return err
		}
		templateData.DockerTemplateData = &dockerTemplateData
		return nil
	case "shell":
		var shellTemplateData ShellTemplateData
		if err := json.Unmarshal(data, &shellTemplateData); err != nil {
			return err
		}
		templateData.ShellTemplateData = &shellTemplateData
		return nil
	case "build-docker":
		var dockerBuildTemplateData DockerBuildTemplateData
		if err := json.Unmarshal(data, &dockerBuildTemplateData); err != nil {
			return err
		}
		templateData.DockerBuildTemplateData = &dockerBuildTemplateData
		return nil
	}
	return fmt.Errorf("unknown type %s", templateData.Type)
}

func (templateData *TemplateData) MarshalJSON() ([]byte, error) {
	switch templateData.Type {
	case "clone":
		return json.Marshal(templateData.CloneTemplateData)
	case "start-docker":
		return json.Marshal(templateData.DockerTemplateData)
	case "shell":
		return json.Marshal(templateData.ShellTemplateData)
	case "build-docker":
		return json.Marshal(templateData.DockerBuildTemplateData)
	}
	return nil, nil
}
