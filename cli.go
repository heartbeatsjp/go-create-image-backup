package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK             = 0
	ExitCodeFlagParseError = 10 + iota
	ExitCodeAWSError
)

// CLI is the command line object.
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
	flags                cliFlags
	mail                 MailClient
}

type cliFlags struct {
	instanceID string
	generation int
	region     string
	service    string
	version    bool
	to         string
	server     string
	port       int
}

// Run invokes the CLI with the given arguments.
func (c *CLI) Run(args []string) int {
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(c.outStream)
	flags.StringVar(&c.flags.instanceID, "instance-id", "", "instance id")
	flags.StringVar(&c.flags.instanceID, "i", "", "instance id(Short)")
	flags.IntVar(&c.flags.generation, "backup-generation", 10, "number of backup generation")
	flags.IntVar(&c.flags.generation, "g", 10, "number of backup generation(Short)")
	flags.StringVar(&c.flags.region, "region", "", "region")
	flags.StringVar(&c.flags.region, "r", "", "region(Short)")
	flags.StringVar(&c.flags.service, "service-tag", "", "value of Service tag")
	flags.StringVar(&c.flags.service, "s", "", "value of Service tag(Short)")
	flags.BoolVar(&c.flags.version, "version", false, "print version information")
	flags.BoolVar(&c.flags.version, "v", false, "print version information(Short)")

	flags.StringVar(&c.flags.to, "mail-to", "", "address to e-mail notification")
	flags.StringVar(&c.flags.to, "t", "", "address to e-mail notification(Short)")
	flags.StringVar(&c.flags.server, "mail-server", "localhost", "address of mail server")
	flags.StringVar(&c.flags.server, "m", "localhost", "address of mail server(Short)")
	flags.IntVar(&c.flags.port, "mail-server-port", 25, "port number of mail server")
	flags.IntVar(&c.flags.port, "p", 25, "port number of mail server(Short)")
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeFlagParseError
	}

	code, err := c.run()
	if err != nil {
		fmt.Fprint(c.errStream, err.Error())
		if c.flags.to != "" && code == ExitCodeAWSError {
			if mailerr := c.mail.Send(c.flags.to, c.flags.server, err.Error(), c.flags.port); mailerr != nil {
				fmt.Fprintf(c.errStream, mailerr.Error())
			}
		}
	}

	return code
}

func (c *CLI) run() (int, error) {
	if c.flags.version {
		fmt.Fprintf(c.outStream, "%s version %s\n", Name, Version)
		return ExitCodeOK, nil
	}

	backup := &Backup{
		InstanceID: c.flags.instanceID,
		Region:     c.flags.region,
		Generation: c.flags.generation,
		Service:    c.flags.service,
		Client:     NewAWSClient(),
	}

	if backup.Region == "" {
		r, err := backup.Client.GetRegion()
		if err != nil {
			return ExitCodeAWSError, fmt.Errorf("failed to get region: %s", err.Error())
		}
		backup.Region = r
	}

	if backup.InstanceID == "" {
		i, err := backup.Client.GetInstanceID()
		if err != nil {
			return ExitCodeAWSError, fmt.Errorf("failed to get instance id: %s", err.Error())
		}
		backup.InstanceID = i
	}

	ctx := context.TODO()

	name, err := backup.Client.GetInstanceName(ctx, backup.InstanceID)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to get instance name: %s", err.Error())
	}
	backup.Name = name

	imageID, err := backup.Create(ctx)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to create backup: %s", err.Error())
	}
	fmt.Fprintf(c.outStream, "create image: %s\n", imageID)

	rotateImageIDs, err := backup.Rotate(ctx, imageID)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to rotate: %s", err.Error())
	}
	fmt.Fprintf(c.outStream, "deregister images: %s\n", strings.Join(rotateImageIDs, ", "))

	return ExitCodeOK, nil
}
