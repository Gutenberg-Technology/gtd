package config

import (
	"os"

	"github.com/joho/godotenv"
)

func ReadTaskEnvFile(filename string) (envMap map[string]string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	return godotenv.Parse(file)
}
