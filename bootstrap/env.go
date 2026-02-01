package bootstrap

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

type Env struct {
	Host             string
	Port             string
	Username         string
	Password         string
	DBName           string
	DBDate           string
	DBSchema         string
	BlobAccountName  string
	BlobAccountKey   string
	ServerAddress    string
	SslMode          string
	ContextTimeout   int
	MobileAppVersion string
	// Connection pool settings for parallel query workloads
	DBMaxOpenConns int
	DBMaxIdleConns int
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
	env.MobileAppVersion = os.Getenv("MOBILE_APP_VERSION")
	// Connection pool settings (defaults optimized for parallel query workloads)
	env.DBMaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", 50)
	env.DBMaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", 25)
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
	log.Printf("MOBILE_APP_VERSION: %s", env.MobileAppVersion)
	return &env

}

// Ensure env file isn't loaded in production -- make sure dockerfile uses production
