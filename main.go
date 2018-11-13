package main

import "os"

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr, mail: MailClient{}}
	os.Exit(cli.Run(os.Args))
}
