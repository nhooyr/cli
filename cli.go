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

// CommandHelp describes the help of a command.
type CommandHelp interface {
	Name() string
	Usage() string
	Desc() string
	Flags(f *flag.FlagSet)
}

// Command represents a command that can be invoked.
type Command interface {
	CommandHelp
	Run(ctx context.Context, args []string) int
}

// Mux is responsible for setting up the routing tree.
type Mux struct {
	leaf Command

	spec CommandHelp
	subs map[string]*Mux
}

// Sub registers a parent command with a sub commands as registered by fn.
func (m *Mux) Sub(spec CommandHelp, fn func(m *Mux)) {
	_, ok := m.subs[spec.Name()]
	if ok {
		panicf("%v is already registered by another command", spec.Name())
	}

	m2 := &Mux{
		spec: spec,
	}

	if m.subs == nil {
		m.subs = make(map[string]*Mux)
	}
	m.subs[spec.Name()] = m2

	fn(m2)
}

// Handle registers the command.
func (m *Mux) Handle(cmd Command) {
	_, ok := m.subs[cmd.Name()]
	if ok {
		panicf("%v is already registered by another command", cmd.Name())
	}

	if m.subs == nil {
		m.subs = make(map[string]*Mux)
	}
	m.subs[cmd.Name()] = &Mux{
		spec: cmd,
		leaf: cmd,
	}
}

// Run starts the CLI.
func Run(ctx context.Context, m Mux) {
	if m.subs == nil && m.leaf == nil {
		panicf("no command registered")
	}
	if len(m.subs) > 1 {
		panicf("cannot register multiple commands on root")
	}

	rootCmd, ok := m.subs[os.Args[0]]
	if !ok {
		panicf("root cmd must be registered on os.Args[0]")
	}

	ctx = context.WithValue(ctx, "fullname", rootCmd.spec.Name())
	status := run(ctx, os.Args[1:], rootCmd)
	os.Exit(status)
}

func run(ctx context.Context, args []string, cmd *Mux) int {
	fullname := ctx.Value("fullname").(string)
	f := initFlagSet(fullname, cmd)

	ctx = context.WithValue(ctx, "usage", f.Usage)

	version := new(bool)
	if fullname == cmd.spec.Name() {
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

	if cmd.leaf != nil {
		return cmd.leaf.Run(ctx, f.Args())
	}

	if len(args) < 1 {
		log.Printf("please provide a subcommand")
		return Help(ctx)
	}

	subcmd, ok := cmd.subs[f.Arg(0)]
	if !ok {
		log.Printf("unknown subcommand: %q", args[0])
		return Help(ctx)
	}

	ctx = context.WithValue(ctx, "fullname", fullname+" "+subcmd.spec.Name())
	return run(ctx, args[1:], subcmd)
}

func initFlagSet(fullname string, cmd *Mux) *flag.FlagSet {
	f := flag.NewFlagSet(fullname, flag.ContinueOnError)
	cmd.spec.Flags(f)

	f.Usage = func() {
		var b bytes.Buffer

		fmt.Fprintf(&b, "usage: %v %v\n", fullname, cmd.spec.Usage())
		fmt.Fprintf(&b, "version: %v\n", Version)

		if cmd.spec.Desc() != "" {
			fmt.Fprintf(&b, "\n%v\n", cmd.spec.Desc())
		}

		var flagsCount int
		f.VisitAll(func(_ *flag.Flag) {
			flagsCount++
		})
		if flagsCount > 0 {
			fmt.Fprintf(&b, "\nflags:\n")
			f.SetOutput(&b)
			f.PrintDefaults()
		}

		if len(cmd.subs) > 0 {
			fmt.Fprintf(&b, "\nsubcommands:\n")

			tw := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			for _, subcmd := range cmd.subs {
				fmt.Fprintf(tw, "  %v %v", subcmd.spec.Name(), subcmd.spec.Usage())
				summary := strings.Split(subcmd.spec.Desc(), "\n")[0]
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

	return f
}

func panicf(f string, v ...interface{}) {
	panic(fmt.Sprintf("cli: "+f, v...))
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
