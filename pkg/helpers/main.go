package helpers

import "strings"

func CanonicalGroup(group string) string {
	return strings.ReplaceAll(group, " ", "_")
}
