package handler

import "fmt"

// format as percent
func FormatAsPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}
