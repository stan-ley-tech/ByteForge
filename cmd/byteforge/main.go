// Command byteforge is the ByteForge binary: it starts the API/web server
// (`byteforge serve`) or runs a collection headlessly (`byteforge run` /
// `byteforge test` / `byteforge export`), depending on the subcommand.
package main

import (
	"fmt"
	"os"

	"github.com/stan-ley-tech/ByteForge/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
