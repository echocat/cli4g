package cli

import (
	_ "embed"
	"strings"
)

//go:embed completion_fish.fish
var fishCompletionScript string

func quoteFish(value string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(value) + "'"
}
