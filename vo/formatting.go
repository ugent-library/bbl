package vo

import "fmt"

func FormatSlice[T any](vals []T) string {
	n := len(vals)
	str := ""
	for i, v := range vals {
		if i > 0 && i == n-1 {
			str += " or "
		} else if i > 0 {
			str += ", "
		}
		str += fmt.Sprint(v)
	}
	return str
}
