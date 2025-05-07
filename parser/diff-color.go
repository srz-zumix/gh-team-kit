package parser

import (
	"strings"

	"github.com/fatih/color"
)

func ColorizeDiff(diff string) string {
	var result string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+ ") {
			result += color.GreenString(line) + "\n"
		} else if strings.HasPrefix(line, "- ") {
			result += color.RedString(line) + "\n"
		} else {
			result += line + "\n"
		}
	}
	return result
}
