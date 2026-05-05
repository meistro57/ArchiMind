// internal/logging/logger.go
package logging

import (
	"log"
	"os"
)

func New() *log.Logger {
	return log.New(os.Stdout, "[ArchiMind] ", log.LstdFlags|log.Lshortfile)
}
