package confirm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ask prints a summary of a pending sensitive operation and blocks until the
// user responds. Returns true only on explicit y/yes. Empty input, EOF, or
// anything else counts as a rejection.
func Ask(op, provider, resource string) bool {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "[!] About to run: %s\n", op)
	if provider != "" {
		fmt.Fprintf(os.Stderr, "    Provider: %s\n", provider)
	}
	if resource != "" {
		fmt.Fprintf(os.Stderr, "    Resource: %s\n", resource)
	}
	fmt.Fprint(os.Stderr, "Proceed? [y/N]: ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
