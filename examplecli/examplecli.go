package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"time"

	"golang.org/x/xerrors"

	"nhooyr.io/cli"
)

func main() {
	log.SetFlags(0)
	ctx := context.Background()
	cli.Run(ctx, &rootCmd{})
}

type rootCmd struct {
	fail int
}

var _ cli.Branch = &rootCmd{}

func (rootCmd *rootCmd) Name() string {
	return "examplecli"
}

func (rootCmd *rootCmd) Desc() string {
	return "My awesome description."
}

func (rootCmd *rootCmd) Flags(f *flag.FlagSet) {
	f.IntVar(&rootCmd.fail, "fail", -1, "Exit with given status.")
}

func (rootCmd *rootCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&lsCmd{
			rootCmd: rootCmd,
		},
	}
}

type lsCmd struct {
	rootCmd *rootCmd
	long    bool
}

var _ cli.Leaf = &lsCmd{}

func (lsCmd *lsCmd) Name() string {
	return "ls"
}

func (lsCmd *lsCmd) Usage() string {
	return "<dir>"
}

func (lsCmd *lsCmd) Desc() string {
	return "My super awesome desc."
}

func (lsCmd *lsCmd) Flags(f *flag.FlagSet) {
	f.BoolVar(&lsCmd.long, "l", false, "Use long format.")
}

func (lsCmd *lsCmd) Run(ctx context.Context, args []string) int {
	if lsCmd.rootCmd.fail != -1 {
		return lsCmd.rootCmd.fail
	}
	if len(args) != 1 {
		return cli.Helpf(ctx, "directory required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ls := exec.CommandContext(ctx, "ls")
	if lsCmd.long {
		ls.Args = append(ls.Args, "-l")
	}
	ls.Args = append(ls.Args, args[0])
	ls.Stdin = os.Stdin
	ls.Stdout = os.Stdout
	ls.Stderr = os.Stderr
	err := ls.Start()
	if err != nil {
		log.Printf("failed to run %q: %v", ls.Args, err)
		return 1
	}

	err = ls.Wait()
	if err != nil {
		cerr := &exec.ExitError{}
		if !xerrors.As(err, &cerr) {
			log.Printf("failed to wait for %q: %v", ls.Args, err)
			return 1
		}
	}

	return 0
}
