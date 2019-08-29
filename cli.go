package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
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
	customTags []Tag
	version    bool
	to         string
	from       string
	server     string
	port       int
}

type tagSliceValue []Tag

func (s *tagSliceValue) String() string {
	var tagStrs []string
	for _, t := range *s {
		tagStrs = append(tagStrs, fmt.Sprintf("%s:%s", t.Key, t.Value))
	}
	return strings.Join(tagStrs, ",")
}

func (s *tagSliceValue) Set(val string) error {
	rawTags := strings.Split(val, ",")

	var tags []Tag
	for _, t := range rawTags {
		if !strings.Contains(t, ":") {
			return errors.New("parse error")
		}

		kv := strings.Split(t, ":")
		if len(kv) != 2 {
			return errors.New("parse error")
		}

		tags = append(tags, Tag{Key: kv[0], Value: kv[1]})
	}

	*s = tagSliceValue(tags)

	return nil
}

func newTagSliceValue(val string, p *[]Tag) *tagSliceValue {
	if val != "" {
		t := (*tagSliceValue)(p)
		if err := t.Set(val); err != nil {
			panic(err)
		}

		return t
	}
	return (*tagSliceValue)(p)
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
	flags.Var(newTagSliceValue("", &c.flags.customTags), "custom-tags", "key-value of Cunstom tags")
	flags.Var(newTagSliceValue("", &c.flags.customTags), "c", "key-value of Cunstom tags(Short)")
	flags.BoolVar(&c.flags.version, "version", false, "print version information")
	flags.BoolVar(&c.flags.version, "v", false, "print version information(Short)")

	flags.StringVar(&c.flags.to, "mail-to", "", "to-address of email notification")
	flags.StringVar(&c.flags.to, "t", "", "to-address of email notification(Short)")
	flags.StringVar(&c.flags.from, "mail-from", "", "from-address of email notification")
	flags.StringVar(&c.flags.from, "f", "", "from-address of email notification(Short)")
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
			from := c.flags.from
			if from == "" {
				from = "go-create-image-backup@localhost.localdomain"
			}
			if mailerr := c.mail.Send(from, c.flags.to, c.flags.server, err.Error(), c.flags.port); mailerr != nil {
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

	sess, err := NewAWSSession()
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("create aws session failed: %s\n", err)
	}

	client, err := NewAWSClient(sess, c.flags.region)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("create aws client failed: %s\n", err)
	}

	backup := &Backup{
		InstanceID: c.flags.instanceID,
		Generation: c.flags.generation,
		Service:    c.flags.service,
		CustomTags: c.flags.customTags,
		Client:     client,
	}

	if backup.InstanceID == "" {
		i, err := backup.Client.GetInstanceID()
		if err != nil {
			return ExitCodeAWSError, fmt.Errorf("failed to get instance id: %s\n", err.Error())
		}
		backup.InstanceID = i
	}

	ctx := context.TODO()

	name, err := backup.Client.GetInstanceName(ctx, backup.InstanceID)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to get instance name: %s\n", err.Error())
	}
	backup.Name = name

	imageID, err := backup.Create(ctx)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to create backup: %s\n", err.Error())
	}
	fmt.Fprintf(c.outStream, "create image: %s\n", imageID)

	rotateImageIDs, err := backup.Rotate(ctx, imageID)
	if err != nil {
		return ExitCodeAWSError, fmt.Errorf("failed to rotate: %s\n", err.Error())
	}
	fmt.Fprintf(c.outStream, "deregister images: %s\n", strings.Join(rotateImageIDs, ", "))

	return ExitCodeOK, nil
}
