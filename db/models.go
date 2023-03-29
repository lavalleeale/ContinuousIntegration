package db

type User struct {
	ID int

	Password string
	Username string

	Organization   Organization `ref:"organization_id" fk:"id" autosave:"true"`
	OrganizationID int
}

type Organization struct {
	ID int

	Repos []Repo `ref:"id" fk:"organization_id"`
	Users []User `ref:"id" fk:"organization_id" autosave:"true"`
}

type Repo struct {
	ID int

	Url string

	Organization   Organization `ref:"organization_id" fk:"id"`
	OrganizationID int

	Builds []Build `ref:"id" fk:"repo_id"`
}

type Build struct {
	ID int

	Repo   Repo `ref:"repo_id" fk:"id"`
	RepoID int

	Containers []Container `ref:"id" fk:"build_id" autosave:"true"`
}

type Container struct {
	ID int

	Name               string
	Code               *int
	Command            string
	Image              string
	Environment        *string
	ServiceCommand     *string
	ServiceImage       *string
	ServiceHealthcheck *string
	ServiceEnvironment *string
	Log                string

	BuildID int
	Build   Build `ref:"build_id" fk:"id"`
}
