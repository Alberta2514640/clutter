package jobsutils

import "strconv"

var AllowedJobTypes = map[string]bool{
	"ansible":   true,
	"terraform": true,
}

func Itoa(i int) string {
	return strconv.Itoa(i)
}
