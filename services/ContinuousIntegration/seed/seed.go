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
	if len(os.Args) > 1 {
		log.Println(auth.Login(os.Args[1], os.Args[2], true))
	}
}
