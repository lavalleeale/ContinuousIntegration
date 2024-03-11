package lib

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/lavalleeale/ContinuousIntegration/lib/db"
)

func TestTemplateData_GetString(t *testing.T) {
	InitTemplates()
	tests := []struct {
		name    string
		fields  TemplateData
		want    string
		wantErr bool
	}{
		{
			name: "clone no config",
			fields: TemplateData{
				TypeSwitch: TypeSwitch{
					Type: "clone",
				},
				CloneTemplateData: &CloneTemplateData{},
			},
			want:    "git  clone test /repo && cd /repo",
			wantErr: false,
		},
		{
			name: "clone with config",
			fields: TemplateData{
				TypeSwitch: TypeSwitch{
					Type: "clone",
				},
				CloneTemplateData: &CloneTemplateData{
					GitConfig: "test",
				},
			},
			want:    "git test clone test /repo && cd /repo",
			wantErr: false,
		},
		{
			name: "start-docker",
			fields: TemplateData{
				TypeSwitch: TypeSwitch{
					Type: "start-docker",
				},
				DockerTemplateData: &DockerTemplateData{},
			},
			want:    "(dockerd > dockerlog.txt 2>&1 &) && until docker version >/dev/null 2>&1; do sleep 1; done",
			wantErr: false,
		},
		{
			name: "shell",
			fields: TemplateData{
				TypeSwitch: TypeSwitch{
					Type: "shell",
				},
				ShellTemplateData: &ShellTemplateData{
					Command: "test",
				},
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "docker build",
			fields: TemplateData{
				TypeSwitch: TypeSwitch{
					Type: "build-docker",
				},
				DockerBuildTemplateData: &DockerBuildTemplateData{
					Tag:      "test",
					CacheTag: "test",
				},
			},
			want: `docker buildx create --driver docker-container --name mybuilder --use --bootstrap && \
docker cp /usr/local/share/ca-certificates/registry.crt buildx_buildkit_mybuilder0:/usr/local/share/ca-certificates/registry.crt &&
docker exec buildx_buildkit_mybuilder0 update-ca-certificates &&
docker login --username=$DOCKER_USER --password=$DOCKER_PASS $REGISTRY &&
docker buildx build --tag test --cache-to=type=registry,ref=$REGISTRY/test,mode=max --cache-from=type=registry,ref=$REGISTRY/test --load images/base`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var templateData TemplateData
			templateString, err := json.Marshal(tt.fields)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Errorf("TemplateData.GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			log.Println(string(templateString))
			if err := json.Unmarshal(templateString, &templateData); err != nil {
				if tt.wantErr {
					return
				}
				t.Errorf("TemplateData.GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := templateData.GetString(db.Repo{Url: "test"})
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateData.GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TemplateData.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}
