package internal

import "strings"

func GetUniqueIdFromArn(arn string) string {
	return arn[strings.LastIndex(arn, "-")+1:]
}
