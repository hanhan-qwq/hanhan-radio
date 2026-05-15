package callback

import (
	"log"
	"os"
)

var Logger = initLogger()

func initLogger() *log.Logger {
	os.MkdirAll("data", 0755)
	f, err := os.OpenFile("data/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.Default()
	}
	log.SetOutput(f)
	return log.Default()
}
