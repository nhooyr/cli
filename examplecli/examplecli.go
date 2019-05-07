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
	var m cli.Mux

	rootCmd := &rootCmd{}
	m.Sub(rootCmd, func(m *cli.Mux) {
		lscmd := &lsCmd{
			rootCmd: rootCmd,
		}

		m.Handle(lscmd)
	})
	cli.Run(ctx, m)
}

type rootCmd struct {
	status int
}

func (rootCmd *rootCmd) Name() string {
	return "examplecli"
}

func (rootCmd *rootCmd) Usage() string {
	return "[-fail] <subcmd>"
}

func (rootCmd *rootCmd) Desc() string {
	return "my awesome description."
}

func (rootCmd *rootCmd) Flags(f *flag.FlagSet) {
	f.IntVar(&rootCmd.status, "fail", 0, "exit with given status")
}

type lsCmd struct {
	rootCmd *rootCmd
	long    bool
}

func (lsCmd *lsCmd) Name() string {
	return "ls"
}

func (lsCmd *lsCmd) Usage() string {
	return "<dir>"
}
func (lsCmd *lsCmd) Desc() string {
	return "my super awesome desc."
}

func (lsCmd *lsCmd) Flags(f *flag.FlagSet) {
	f.BoolVar(&lsCmd.long, "l", false, "long declaration")
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
