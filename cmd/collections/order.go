package collections

import "strings"

var sortFields = []string{
	"title",
	"shortname",
	"start_date",
	"platform",
	"instrument",
	"sensor",
	"provider",
	"revsion_date",
}

func validSortField(val string) bool {
	val = strings.ReplaceAll(val, "-", "")
	for _, s := range sortFields {
		if val == s {
			return true
		}
	}
	return false
}
