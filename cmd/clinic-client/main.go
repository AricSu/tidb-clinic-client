package main

import (
	cli "github.com/AricSu/tidb-clinic-client/internal/cli"
	"log"
)

func main() {
	if err := cli.NewCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
