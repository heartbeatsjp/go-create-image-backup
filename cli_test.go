package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_customTagsFlag(t *testing.T) {
	var cases = []struct {
		args string
		want int
	}{
		{
			args: "go-create-image-backup -custom-tags tag",
			want: ExitCodeFlagParseError,
		},
		{
			args: "go-create-image-backup -custom-tags tag:val1:val2",
			want: ExitCodeFlagParseError,
		},
		{
			args: "go-create-image-backup -custom-tags ,tag:val",
			want: ExitCodeFlagParseError,
		},
		{
			args: "go-create-image-backup -custom-tags tag:val,",
			want: ExitCodeFlagParseError,
		},
		{
			args: "go-create-image-backup -custom-tags tag:val,tag",
			want: ExitCodeFlagParseError,
		},
		{
			args: "go-create-image-backup -custom-tags",
			want: ExitCodeFlagParseError,
		},
	}

	for _, c := range cases {
		t.Run(c.args, func(t *testing.T) {
			cli := &CLI{outStream: new(bytes.Buffer), errStream: new(bytes.Buffer)}
			got := cli.Run(strings.Split(c.args, " "))
			if c.want != got {
				t.Errorf("want %d, got %d", c.want, got)
			}
		})
	}
}
