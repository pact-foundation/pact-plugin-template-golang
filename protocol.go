package main

import (
	"fmt"
	"regexp"
	"strings"
)

func generateMattMessage(message string) string {
	return fmt.Sprintf("MATT%sMATT", message)
}

func parseMattMessage(message string) string {
	return strings.TrimSpace(strings.ReplaceAll(message, "MATT", ""))
}

func isValidMessage(str string) bool {
	matched, err := regexp.MatchString(`^MATT.*MATT$`, str)
	if err != nil {
		return false
	}

	return matched
}
