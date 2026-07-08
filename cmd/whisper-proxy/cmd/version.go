package cmd

import (
	"fmt"
	"strings"
)

var Version = "dev"
var Commit = "none"
var BuildTime = "unknown"

func FullVersionString() string {
	format := `
		VERSION: %s
		COMMIT:  %s
		BUILT:   %s
	`
	format = strings.ReplaceAll(format, "\t", "")
	format = strings.TrimSpace(format)

	return fmt.Sprintf(format, Version, Commit, BuildTime)
}
