package bootstrap

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Env struct {
	AppEnv         string
	Host           string
	Port           string
	Username       string
	Password       string
	DBName         string
	ServerAddress  string
	ContextTimeout int
}

func NewEnv() *Env {
	env := Env{}
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	env.Host = os.Getenv("DB_HOST")
	env.Port = os.Getenv("DB_PORT")
	env.Username = os.Getenv("DB_USERNAME")
	env.Password = os.Getenv("DB_PASSWORD")
	env.DBName = os.Getenv("DB_NAME")

	if env.AppEnv == "development" {
		log.Println("The App is running in development mode")
	}

	return &env

}
