package db

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Db *gorm.DB

func Open() error {
	var err error
	Db, err = gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println(Db.AutoMigrate(&User{}, &Organization{}, &Repo{}, &Build{}, &Container{}, &ServiceContainer{},
		&ContainerGraphEdge{}, &UploadedFile{}, &OrganizationInvite{}))

	return nil
}
