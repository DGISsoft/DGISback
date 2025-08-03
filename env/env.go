package env

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func GetEnv[T any](nameEnv string, defaultValue T) T {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	valueStr := os.Getenv(nameEnv)

	var value any

	switch any(defaultValue).(type) {
	case int:
		v,err :=strconv.Atoi(valueStr)
		if err != nil {
			log.Fatal("Error converting string to int",nameEnv, err)
			return defaultValue
		}
		value = v
	case bool:
		v,err := strconv.ParseBool(valueStr)
		if err != nil {
			log.Fatal("Error converting string to bool",nameEnv, err)
			return defaultValue
		}
		value = v
	case float64:
		v,err := strconv.ParseFloat(valueStr,64)
		if err != nil {
			log.Fatal("Error converting string to float64",nameEnv, err)
			return defaultValue
		}
		value = v
	default:
		value = valueStr
	}

	return value.(T)
}