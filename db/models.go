package db

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	Password string
	Username string `gorm:"primaryKey"`

	Organization   Organization
	OrganizationID uint
}

type Organization struct {
	ID uint

	Repos []Repo `gorm:"constraint:OnDelete:CASCADE;"`
	Users []User `gorm:"constraint:OnDelete:CASCADE;"`
}

type Repo struct {
	ID uint

	Url string

	Organization   Organization
	OrganizationID uint

	Builds []Build `gorm:"constraint:OnDelete:CASCADE;"`
}

type Build struct {
	ID        uint
	GitConfig string

	CreatedAt time.Time
	Repo      Repo
	RepoID    uint

	Containers []Container `gorm:"constraint:OnDelete:CASCADE;"`
}

type Container struct {
	Id uint `gorm:"primaryKey"`

	Name               string
	Code               *int
	Command            string `gorm:"size:8192"`
	Image              string
	Environment        *string `gorm:"size:512"`
	ServiceCommand     *string `gorm:"size:512"`
	ServiceImage       *string
	ServiceHealthcheck *string `gorm:"size:512"`
	ServiceEnvironment *string `gorm:"size:512"`
	Log                string  `gorm:"size:100000"`

	UploadedFiles []UploadedFile `gorm:"constraint:OnDelete:CASCADE;"`

	NeededFiles []NeededFile `gorm:"constraint:OnDelete:CASCADE;"`

	EdgesToward []ContainerGraphEdge `gorm:"constraint:OnDelete:CASCADE;foreignKey:ToID"`
	EdgesFrom   []ContainerGraphEdge `gorm:"constraint:OnDelete:CASCADE;foreignKey:FromID"`

	BuildID uint
	Build   Build
}

type NeededFile struct {
	Id          uint `gorm:"primaryKey"`
	From        string
	FromPath    string
	ContainerID uint
	Container   Container
}

func (v Container) ID() string {
	return v.Name
}

type UploadedFile struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;"`
	Path        string
	Container   Container
	ContainerID uint
	Bytes       []byte
}

func (file *UploadedFile) BeforeCreate(tx *gorm.DB) (err error) {
	if file.ID.Variant() == uuid.Reserved {
		file.ID = uuid.New()
	}
	return
}

type ContainerGraphEdge struct {
	ID uint

	ToID   uint
	FromID uint
	From   Container `gorm:"foreignKey:FromID"`
	To     Container `gorm:"foreignKey:ToID"`
}
