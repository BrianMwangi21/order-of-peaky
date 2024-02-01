package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetBinanceKeys() (string, string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return os.Getenv("BINANCE_API_KEY"), os.Getenv("BINANCE_SECRET_KEY")
}
