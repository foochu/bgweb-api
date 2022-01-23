package gnubg

import "fmt"

func logWarning(msg string) {
	fmt.Printf("WARN: %v\n", msg)
}

func logWarningf(format string, a ...interface{}) {
	logWarning(fmt.Sprintf(format, a...))
}

// func logError(err error) {
// 	fmt.Printf("ERROR: %v\n", err)
// }
