package speedtest

import (
	"log"
	"os"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}