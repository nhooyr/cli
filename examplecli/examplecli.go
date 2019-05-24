package main

import (
	"context"
	"flag"
	"golang.org/x/xerrors"
	"log"
	"nhooyr.io/cli"
	"os"
	"os/exec"
	"time"
)

func main() {
	log.SetFlags(0)
	ctx := context.Background()
	var m cli.Tree

	rootCmd := &rootCmd{}
	m.Branch(rootCmd)
	cli.Run(ctx, m)
}

type rootCmd struct {
	status int
}

func (rootCmd *rootCmd) Name() string {
	return "examplecli"
}

func (rootCmd *rootCmd) ArgsHelp() string {
	return ""
}

func (rootCmd *rootCmd) Desc() string {
	return "My awesome description."
}

func (rootCmd *rootCmd) Flags(f *flag.FlagSet) {
	f.IntVar(&rootCmd.status, "fail", 0, "Exit with given status.")
}

func (rootCmd *rootCmd) Branch(t cli.Tree) {
	lscmd := &lsCmd{
		name:    "install-for-chrome-ext",
		rootCmd: rootCmd,
	}
	lscmd2 := &lsCmd{
		name:    "ls",
		rootCmd: rootCmd,
	}

	t.Leaf(lscmd)
	t.Leaf(lscmd2)
}

type lsCmd struct {
	name    string
	rootCmd *rootCmd
	long    bool
}

func (lsCmd *lsCmd) Name() string {
	return lsCmd.name
}

func (lsCmd *lsCmd) ArgsHelp() string {
	return "<dir>"
}

func (lsCmd *lsCmd) Desc() string {
	return "My super awesome desc."
}

func (lsCmd *lsCmd) Flags(f *flag.FlagSet) {
	if lsCmd.name != "ls" {
		return
	}
	f.BoolVar(&lsCmd.long, "l", false, "Long listing.")
}

func (lsCmd *lsCmd) Run(ctx context.Context, args []string) int {
	if len(args) != 1 {
		log.Println("you must provide a single argument")
		return cli.Help(ctx)
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
		log.Println("failed to run %q: %v", ls.Args, err)
		return 1
	}

	err = ls.Wait()
	if err != nil {
		cerr := &exec.ExitError{}
		if !xerrors.As(err, &cerr) {
			log.Println("failed to wait for %q: %v", ls.Args, err)
			return 1
		}
	}

	return 0
}
