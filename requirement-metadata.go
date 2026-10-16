package cli

import "strings"

func parseNameAndDescription(values []string) (name, description string) {
	if len(values) == 0 {
		return "", ""
	}
	return values[0], strings.Join(values[1:], " ")
}
