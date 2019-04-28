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

type Spec struct {
	Usage string
	Desc  string
	Flags flag.FlagSet
}

type ParentCommand interface {
	Spec() Spec
}

type Command interface {
	Run(ctx context.Context, args []string) int
	Spec() Spec
}

// Mux is a router for your CLI.
type Mux struct{}

func (m Mux) Handle(name string, cmd Command) {

}

func (m Mux) Sub(name string, pcmd ParentCommand, fn func(m Mux)) {

}

func Run(ctx context.Context, m Mux) {

}

func (subcmds Mux) Run(ctx context.Context, args []string) int {
	if len(args) < 1 {
		log.Printf("please provide a subcommand")
		return Help(ctx)
	}

	for _, subspec := range subcmds {
		if subspec.Name == args[0] {
			return run(ctx, args[1:], subspec)
		}
	}

	log.Printf("unknown subcommand: %q", args[0])
	return Help(ctx)
}

// Version represents the git tag/revision for this build./
// Please set this as you see fit.
// I would recommend a go generated version file or injecting
// the value into go build.
var Version = "<dev>"

// Help prints the usage for the selected command.
// The passed context should be derived from the context
// passed to the handler.
func Help(ctx context.Context) int {
	ctx.Value("usage").(func())()
	return 1
}

// Run begins the CLI with the given root command.
func Run(ctx context.Context, cmd Command) {
	ctx = context.WithValue(ctx, "fullname", os.Args[0])
	status := run(ctx, os.Args[1:], cmd)
	os.Exit(status)
}

func run(ctx context.Context, args []string, spec Command) int {
	fullname := ctx.Value("fullname").(string)
	f := initFlagSet(fullname, spec)

	version := new(bool)
	if fullname == spec.Name {
		version = f.Bool("version", false, "Print version and exit.")
	}

	err := f.Parse(args)
	if err != nil {
		return 1
	}

	if *version || f.Arg(0) == "--version" || f.Arg(0) == "-version" {
		os.Stdout.WriteString(Version + "\n")
		return 0
	}

	ctx = context.WithValue(ctx, "usage", f.Usage)
	ctx = context.WithValue(ctx, "fullname", fullname)
	return spec.Handler.Run(ctx, f.Args())
}

func initFlagSet(fullname string, spec Command) *flag.FlagSet {
	spec.Flags.Init(fullname, flag.ContinueOnError)

	spec.Flags.Usage = func() {
		var b bytes.Buffer

		fmt.Fprintf(&b, "usage: %v %v\n", fullname, spec.Usage)
		fmt.Fprintf(&b, "version: %v\n", Version)

		if spec.Desc != "" {
			fmt.Fprintf(&b, "\n%v\n", spec.Desc)
		}

		var flagsCount int
		spec.Flags.VisitAll(func(_ *flag.Flag) {
			flagsCount++
		})
		if flagsCount > 0 {
			fmt.Fprintf(&b, "\nflags:\n")
			spec.Flags.SetOutput(&b)
			spec.Flags.PrintDefaults()
		}

		subcmds, ok := spec.Handler.(Mux)
		if ok {
			fmt.Fprintf(&b, "\nsubcommands:\n")

			tw := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			for _, subcmd := range subcmds {
				fmt.Fprintf(tw, "  %v %v", subcmd.Name, subcmd.Usage)
				summary := strings.Split(subcmd.Desc, "\n")[0]
				if summary != "" {
					fmt.Fprintf(tw, "\t%v", summary)
				}
				fmt.Fprintf(tw, "\n")
			}
			err := tw.Flush()
			if err != nil {
				panic(fmt.Sprintf("cli: tabwriter flush error: %v", err))
			}
		}

		os.Stderr.Write(b.Bytes())
	}

	return &spec.Flags
}
