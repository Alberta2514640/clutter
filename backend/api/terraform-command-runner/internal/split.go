package internal

import "strings"

func SplitAndTrim(input string) []string {
	res := []string{}
	for _, s := range strings.Split(input, ",") {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
