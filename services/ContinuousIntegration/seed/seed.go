package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/auth"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	err = db.Open()

	if err != nil {
		log.Fatal("Failed to Open DB")
	}

	log.Println(db.Db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&db.Organization{}).RowsAffected)
	for _, seq := range []string{
		"repos_id_seq", "builds_id_seq",
		"container_graph_edges_id_seq", "needed_files_id_seq", "service_containers_id_seq",
		"containers_id_seq",
	} {
		db.Db.Exec("ALTER SEQUENCE " + seq + " RESTART WITH 1")
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "default":
			auth.Login("tester", "tester", true)
		case "invalidAccess":
			auth.Login("user1", "user1", true)
			db.Db.Create(&db.Container{Build: db.Build{
				Repo: db.Repo{OrganizationID: "user1", Url: "http://test"},
			}})
			auth.Login("user2", "user2", true)
		}
	}
}
