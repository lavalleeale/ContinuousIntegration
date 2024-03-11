package db

import (
	"time"

	"github.com/google/uuid"
	pq "github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	Password string
	Username string `gorm:"primaryKey"`

	InstallationIds pq.Int64Array `gorm:"type:integer[]"`

	Organization   Organization
	OrganizationID string
}

type Organization struct {
	ID string `gorm:"primaryKey"`

	Repos []Repo `gorm:"constraint:OnDelete:CASCADE;"`
	Users []User `gorm:"constraint:OnDelete:CASCADE;"`
}

type Repo struct {
	ID uint

	Url            string
	InstallationId *int64
	GithubRepoId   *int64

	Organization   Organization
	OrganizationID string

	Builds []Build `gorm:"constraint:OnDelete:CASCADE;"`
}

type Build struct {
	ID        uint
	GitConfig string
	Status    string `gorm:"default:'pending'"`

	CreatedAt time.Time
	Repo      Repo
	RepoID    uint

	Containers []Container `gorm:"constraint:OnDelete:CASCADE;"`

	UploadedFiles []UploadedFile `gorm:"constraint:OnDelete:CASCADE;"`
}

type Container struct {
	Name        string `gorm:"primaryKey;index:idx_name,unique"`
	Code        *int
	Command     string `gorm:"size:8192"`
	Image       string
	Environment *string `gorm:"size:512"`
	Log         string  `gorm:"size:100000"`
	Persist     *string

	ServiceContainers []ServiceContainer `gorm:"constraint:OnDelete:CASCADE;foreignKey:ContainerName,BuildID;references:Name,BuildID"`

	FilesUploaded []UploadedFile `gorm:"constraint:OnDelete:CASCADE;foreignKey:FromName,BuildID;references:Name,BuildID"`

	FilesNeeded []UploadedFile `gorm:"many2many:files_needed;constraint:OnDelete:CASCADE;"`

	EdgesToward []ContainerGraphEdge `gorm:"constraint:OnDelete:CASCADE;foreignKey:ToName,BuildID;references:Name,BuildID"`
	EdgesFrom   []ContainerGraphEdge `gorm:"constraint:OnDelete:CASCADE;foreignKey:FromName,BuildID;references:Name,BuildID"`

	BuildID uint `gorm:"primaryKey;index:idx_name,unique"`
	Build   Build
}

type ServiceContainer struct {
	Id uint `gorm:"primaryKey"`

	Name        string
	Image       string
	Healthcheck string  `gorm:"size:512"`
	Command     *string `gorm:"size:512"`
	Environment *string `gorm:"size:512"`

	ContainerName string
	BuildID       uint
}

func (v Container) ID() string {
	return v.Name
}

type UploadedFile struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;"`

	Path  string
	Bytes []byte

	FromName string
	From     Container    `gorm:"foreignKey:FromName,BuildID;references:Name,BuildID"`
	To       []*Container `gorm:"many2many:files_needed;"`

	BuildID uint
	Build   Build
}

func (file *UploadedFile) BeforeCreate(tx *gorm.DB) (err error) {
	if file.ID.Variant() == uuid.Reserved {
		file.ID = uuid.New()
	}
	return
}

type ContainerGraphEdge struct {
	ID uint

	BuildID uint

	ToName   string
	FromName string
	From     Container `gorm:"foreignKey:FromName,BuildID;references:Name,BuildID"`
	To       Container `gorm:"foreignKey:ToName,BuildID;references:Name,BuildID"`
}
