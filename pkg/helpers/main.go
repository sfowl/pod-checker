package helpers

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func CanonicalGroup(group string) string {
	return strings.ReplaceAll(group, " ", "_")
}

func CheckFileExist(file string, msg string) bool {
	if _, err := os.Stat(file); err == nil {
		log.Warnf(msg)
		return true
	}
	return false

}
