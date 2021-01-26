package debug

import (
	"fmt"
	"os"
)

func DPrintln(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

func DPrintf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}
