package logger

import (
	"os"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

func InitLogger() {
	if os.Getenv("FORCE_LOG_COLORS") == "true" || isatty.IsTerminal(os.Stdout.Fd()) {
		log.SetFormatter(&log.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		})
	} else {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
	}
}
