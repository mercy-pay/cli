// Command mercypay is the Mercy Pay command line.
package main

import (
	"os"

	"github.com/mercy-pay/cli/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
