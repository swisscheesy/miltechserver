package bootstrap

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	Host            string
	Port            string
	Username        string
	Password        string
	DBName          string
	DBDate          string
	DBSchema        string
	BlobAccountName string
	BlobAccountKey  string
	ServerAddress   string
	SslMode         string
	ContextTimeout  int
}

func NewEnv() *Env {
	env := Env{}
	var err error
	if os.Getenv("DEBUG") == "true" {
		log.Println("Debug build: Loading .env file")
		err = godotenv.Load(".env")
		env.SslMode = "disable"
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		log.Println("Production build: Skipping .env file")
		env.SslMode = "require"
	}
	// Database
	env.Host = os.Getenv("DB_HOST")
	env.Port = os.Getenv("DB_PORT")
	env.Username = os.Getenv("DB_USERNAME")
	env.Password = os.Getenv("DB_PASSWORD")
	env.DBName = os.Getenv("DB_NAME")
	env.DBDate = os.Getenv("DB_DATE")
	env.DBSchema = os.Getenv("DB_SCHEMA")
	// Blob Storage
	env.BlobAccountName = os.Getenv("BLOB_ACCOUNT_NAME")
	env.BlobAccountKey = os.Getenv("BLOB_ACCOUNT_KEY")

	log.Printf("DB_HOST: %s", env.Host)
	log.Printf("DB_PORT: %s", env.Port)
	log.Printf("DB_USERNAME: %s", env.Username)
	log.Printf("DB_NAME: %s", env.DBName)
	log.Printf("DB_DATE: %s", env.DBDate)
	log.Printf("DB_SCHEMA: %s", env.DBSchema)
	log.Printf("SSL_MODE: %s", env.SslMode)
	return &env

}

// Ensure env file isn't loaded in production -- make sure dockerfile uses production
