// Package cli provides a minimal API for implementing user friendly command
// line programs in Go.
package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

// Version represents the version of the CLI.
// You can use go generate or go build -X to populate this.
var Version = "<dev>"

// Command represents a CLI command.
// Any type that implements Command must implement either Leaf or Branch.
type Command interface {
	// Name returns the name a user will use to refer to the command.
	Name() string

	// Desc returns a description for the command.
	// The first sentence will be used in the help for
	// the parent command so try to make sure its short.
	Desc() string

	// Flags should register the command's flags on the passed flagset.
	Flags(f *flag.FlagSet)
}

// Leaf represents a command that can be invoked.
type Leaf interface {
	Command

	// Usage returns a string that describes the command's args.
	// A flags field will be added automatically to the usage line when
	// at least one flag is defined.
	Usage() string

	// Run is called when the command is invoked.
	// The returned integer is the status code for the command.
	Run(ctx context.Context, args []string) int
}

// Branch represents a command that has subcommands.
type Branch interface {
	Command

	// Subcommands returns the command's subcommands.
	Subcommands() []Command
}

// Helpf prints the msg followed by the help for the
// current command.
//
// The passed context must be derived from the context
// passed to Run.
func Helpf(ctx context.Context, msg string, v ...interface{}) int {
	log.Printf(msg+"\n\n", v...)
	ctx.Value(usageKey{}).(func())()
	return 1
}

// Run begins the CLI with cmd.
func Run(ctx context.Context, cmd Command) {
	ctx = context.WithValue(ctx, fullnameKey{}, cmd.Name())
	status := run(ctx, os.Args[1:], cmd)
	os.Exit(status)
}

func run(ctx context.Context, args []string, cmd Command) int {
	fullname := ctx.Value(fullnameKey{}).(string)
	f := initFlagSet(fullname, cmd)

	ctx = context.WithValue(ctx, usageKey{}, f.Usage)

	version := new(bool)
	if fullname == cmd.Name() {
		version = f.Bool("version", false, "Print version and exit.")
	}

	err := f.Parse(args)
	if err != nil {
		return 1
	}

	if *version {
		os.Stdout.WriteString(Version + "\n")
		return 0
	}

	switch cmd := cmd.(type) {
	case Leaf:
		return cmd.Run(ctx, f.Args())
	case Branch:
		if f.NArg() < 1 {
			return Helpf(ctx, "please provide a subcommand")
		}

		for _, subcmd := range cmd.Subcommands() {
			if subcmd.Name() == f.Arg(0) {
				ctx = context.WithValue(ctx, fullnameKey{}, fullname+" "+subcmd.Name())
				return run(ctx, f.Args()[1:], subcmd)
			}
		}

		return Helpf(ctx, "unknown subcommand: %q", f.Arg(0))
	default:
		panicf("cmd %T does not implement cli.Leaf or cli.Branch", cmd)
		panic("unreachable")
	}
}

func usage(cmd Command, f *flag.FlagSet) string {
	usage := ""

	appendUsage := func(str string) {
		if str == "" {
			return
		}

		if usage != "" {
			usage += " "
		}
		usage += str
	}

	if countFlags(f) > 0 {
		appendUsage("[flags...]")
	}

	switch cmd := cmd.(type) {
	case Leaf:
		appendUsage(cmd.Usage())
	case Branch:
		appendUsage("<subcmd>")
	}

	return usage
}

func countFlags(f *flag.FlagSet) int {
	var flagsCount int
	f.VisitAll(func(_ *flag.Flag) {
		flagsCount++
	})
	return flagsCount
}

func initFlagSet(fullname string, cmd Command) *flag.FlagSet {
	f := flag.NewFlagSet(fullname, flag.ContinueOnError)
	cmd.Flags(f)

	f.Usage = func() {
		var b bytes.Buffer

		fmt.Fprintf(&b, "Usage:\n\t%v %v\n", fullname, usage(cmd, f))

		if fullname == cmd.Name() {
			fmt.Fprintf(&b, "\nVersion: %v\n", Version)
		}

		if cmd.Desc() != "" {
			fmt.Fprintf(&b, "\n%v\n", cmd.Desc())
		}

		if countFlags(f) > 0 {
			fmt.Fprintf(&b, "\nFlags:\n")
			f.SetOutput(&b)
			f.PrintDefaults()
		}

		if cmd, ok := cmd.(Branch); ok {
			fmt.Fprintf(&b, "\nSubcommands:\n")

			tw := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			for _, subcmd := range cmd.Subcommands() {
				f2 := flag.NewFlagSet(fullname+" "+subcmd.Name(), flag.ContinueOnError)
				subcmd.Flags(f2)
				fmt.Fprintf(tw, "  %v\t%v", subcmd.Name(), usage(subcmd, f2))
				summary := strings.Split(subcmd.Desc(), "\n")[0]
				if summary != "" {
					fmt.Fprintf(tw, "\t%v", summary)
				}
				fmt.Fprintf(tw, "\n")
			}
			err := tw.Flush()
			if err != nil {
				panicf("tabwriter flush error: %v", err)
			}
		}

		os.Stderr.Write(b.Bytes())
	}

	return f
}

func panicf(f string, v ...interface{}) {
	panic(fmt.Sprintf("cli: "+f, v...))
}

type (
	usageKey struct{}
	fullnameKey struct{}
)
