package main

import (
	"log"
	"os"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DATABASE_URL")
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalln(err)
	}

	queries := []string{
		"ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_email TEXT DEFAULT 'contact@eraya.com';",
		"ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_phone TEXT DEFAULT '+880 1234 567890';",
		"ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_address TEXT DEFAULT 'Dhaka, Bangladesh';",
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Printf("Error executing query %s: %v", q, err)
		} else {
			log.Printf("Successfully executed query: %s", q)
		}
	}
}
