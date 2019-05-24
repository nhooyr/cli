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

type spec interface {
	Name() string
	Desc() string
	Flags(f *flag.FlagSet)
}

// Branch is a command with subcommands.
type Branch interface {
	Name() string
	Desc() string
	Flags(f *flag.FlagSet)

	// Branch registers the branch's subcommands.
	Branch(m Tree)
}

// Leaf is a command that can be invoked.
type Leaf interface {
	Name() string

	// Usage returns a string indicating the arguments that can
	// be passed to the command. E.g. "<src> <dest>" or "hour:minute:second".
	// It is used by the autogenerated help.
	Usage() string

	Desc() string
	Flags(f *flag.FlagSet)

	// Run is called when the command is invoked.
	// The returned integer is the status code for the command.
	Run(ctx context.Context, args []string) int
}

// Tree represents the CLI tree.
type Tree struct {
	leaf Leaf
	spec spec
	subs map[string]Tree
}

// Branch registers a branch.
func (m *Tree) Branch(branch Branch) {
	_, ok := m.subs[branch.Name()]
	if ok {
		panicf("%v is already registered by another command", branch.Name())
	}

	m2 := Tree{
		spec: branch,
		subs: make(map[string]Tree),
	}

	if m.subs == nil {
		m.subs = make(map[string]Tree)
	}
	m.subs[branch.Name()] = m2

	branch.Branch(m2)

	if len(m2.subs) == 0 {
		panicf("branch command %v must register at least one command", branch.Name())
	}
}

// Leaf registers a leaf command.
func (m *Tree) Leaf(leaf Leaf) {
	_, ok := m.subs[leaf.Name()]
	if ok {
		panicf("%v is already registered by another command", leaf.Name())
	}

	if m.subs == nil {
		m.subs = make(map[string]Tree)
	}
	m.subs[leaf.Name()] = Tree{
		spec: leaf,
		leaf: leaf,
	}
}

// Run starts the CLI with the given tree.
func Run(ctx context.Context, m Tree) {
	if m.subs == nil && m.leaf == nil {
		panicf("no command registered")
	}
	if len(m.subs) > 1 {
		panicf("cannot register multiple commands on root")
	}

	var rootCmd Tree
	for _, rootCmd = range m.subs {
	}

	ctx = context.WithValue(ctx, "fullname", rootCmd.spec.Name())
	status := run(ctx, os.Args[1:], rootCmd)
	os.Exit(status)
}

func run(ctx context.Context, args []string, cmd Tree) int {
	fullname := ctx.Value("fullname").(string)
	f := initFlagSet(fullname, cmd)

	ctx = context.WithValue(ctx, usageKey{}, f.Usage)

	version := new(bool)
	if fullname == cmd.spec.Name() {
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

func usage(cmd Tree, flagCount int) string {
	usage := ""

	if flagCount > 0 {
		usage += "[flags] "
	}

	if cmd.leaf != nil {
		usage += cmd.leaf.Usage()
	} else {
		usage += "<subcmd> "
	}

	return strings.TrimSpace(usage)
}

func countFlags(f *flag.FlagSet) int {
	var flagsCount int
	f.VisitAll(func(_ *flag.Flag) {
		flagsCount++
	})
	return flagsCount
}

func initFlagSet(fullname string, cmd Tree) *flag.FlagSet {
	f := flag.NewFlagSet(fullname, flag.ContinueOnError)
	cmd.spec.Flags(f)

	f.Usage = func() {
		var b bytes.Buffer

		flagsCount := countFlags(f)
		fmt.Fprintf(&b, "usage: %v %v\n", fullname, usage(cmd, flagsCount))
		fmt.Fprintf(&b, "version: %v\n", Version)

		if cmd.spec.Desc() != "" {
			fmt.Fprintf(&b, "\n%v\n", cmd.spec.Desc())
		}

		if flagsCount > 0 {
			fmt.Fprintf(&b, "\nflags:\n")
			f.SetOutput(&b)
			f.PrintDefaults()
		}

		if len(cmd.subs) > 0 {
			fmt.Fprintf(&b, "\nsubcommands:\n")

			tw := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			for _, subcmd := range cmd.subs {
				f2 := flag.NewFlagSet(fullname+" "+subcmd.spec.Name(), flag.ContinueOnError)
				subcmd.spec.Flags(f2)
				fmt.Fprintf(tw, "  %v\t%v", subcmd.spec.Name(), usage(subcmd, countFlags(f2)))
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

// Version represents the git tag/revision for this build.
// You can use go generate or go build to populate this.
// Examples soon.
var Version = "<dev>"

type usageKey struct{}

// Help prints the usage for the selected command.
// The passed context should be derived from the context
// passed to the handler.
func Help(ctx context.Context) int {
	ctx.Value(usageKey{}).(func())()
	return 1
}
