package tool

import "strings"

func SplitReadinessConditionReason(reason string) (setting, present, pid string) {
	v := strings.Split(reason, "/")
	switch len(v) {
	case 1:
		return v[0], "", ""
	case 2:
		return v[0], v[1], ""
	case 3:
		return v[0], v[1], v[2]
	default:
		return "Unknown", "Unknown", ""
	}
}
