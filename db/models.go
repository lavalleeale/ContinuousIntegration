package db

type User struct {
	Password string
	Username string `gorm:"primaryKey"`

	Organization   Organization
	OrganizationID int
}

type Organization struct {
	ID int

	Repos []Repo
	Users []User
}

type Repo struct {
	ID int

	Url string

	Organization   Organization
	OrganizationID int

	Builds []Build
}

type Build struct {
	ID int

	Repo   Repo
	RepoID int

	Containers []Container
}

type Container struct {
	ID int

	Name               string
	Code               *int
	Command            string `gorm:"size:8192"`
	Image              string
	Environment        *string `gorm:"size:512"`
	ServiceCommand     *string `gorm:"size:512"`
	ServiceImage       *string
	ServiceHealthcheck *string `gorm:"size:512"`
	ServiceEnvironment *string `gorm:"size:512"`
	Log                string  `gorm:"size:25000"`

	BuildID int
	Build   Build
}
