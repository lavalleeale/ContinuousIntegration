package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/services/ContinuousIntegration/db"
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

	db.Db.Where("1=1").Delete(&db.Organization{})
}
