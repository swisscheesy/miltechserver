package bootstrap

import (
	"github.com/joho/godotenv"
	"log"
)

type Env struct {
	AppEnv         string
	ServerAddress  string
	ContextTimeout int
}

func NewEnv() *Env {
	env := Env{}
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if env.AppEnv == "development" {
		log.Println("The App is running in development mode")
	}

	return &env

}
