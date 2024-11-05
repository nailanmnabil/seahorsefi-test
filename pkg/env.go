package pkg

import (
	"log"

	"github.com/joho/godotenv"
)

func GetEnv(key, defaultValue string) string {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		log.Fatalf("Failed to read .env file: %v\n", err)
	}

	value := envFile[key]
	if value == "" {
		if defaultValue == "" {
			log.Fatalf("Failed to get env")
		}
		return defaultValue
	}
	return value
}
