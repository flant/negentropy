package util

func FirstNonEmptyString(args ...string) string {
	for _, arg := range args {
		if arg != "" {
			return arg
		}
	}
	return ""
}
