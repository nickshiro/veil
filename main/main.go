package main

import (
	"flag"
	"io"
	"os"

	"github.com/nickshiro/veil/main/commands/base"
	_ "github.com/nickshiro/veil/main/distro/all"
)

func main() {
	os.Args = getArgsV4Compatible()

	base.RootCommand.Long = "Xray is a platform for building proxies."
	base.RootCommand.Commands = append(
		[]*base.Command{
			cmdRun,
			cmdVersion,
		},
		base.RootCommand.Commands...,
	)
	base.Execute()
}

func getArgsV4Compatible() []string {
	if len(os.Args) == 1 {
		return []string{os.Args[0], "run"}
	}
	if os.Args[1][0] != '-' {
		return os.Args
	}
	version := false
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.BoolVar(&version, "version", false, "")
	// parse silently, no usage, no error output
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	err := fs.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return []string{os.Args[0], "help"}
	}
	if version {
		return []string{os.Args[0], "version"}
	}
	return append([]string{os.Args[0], "run"}, os.Args[1:]...)
}
