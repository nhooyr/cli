package main

import (
	"context"
	"golang.org/x/xerrors"
	"log"
	"nhooyr.io/cli"
	"os"
	"os/exec"
	"time"
)

func main() {
	log.SetFlags(0)

	r := &rootCmd{}

	var m cli.Mux
	m.Sub(os.Args[0], r, func(m cli.Mux) {
		m.Handle("ls", &lsCmd{
			rootCmd: r,
		})
		// .. other subcommands would be defined in a similar manner.
	})
}

type rootCmd struct {
	status int
}

func (rootCmd *rootCmd) Spec() cli.Spec {
	spec := cli.Spec{
		Usage: "[-fail] <SUBCMD>",
		Desc:  "my awesome description.",
	}
	spec.Flags.IntVar(&rootCmd.status, "fail", 0, "exit with given status")
	return spec
}

type lsCmd struct {
	rootCmd *rootCmd
	long    bool
}

func (lsCmd *lsCmd) Spec() cli.Spec {
	s := cli.Spec{
		Usage: "<DIR>",
		Desc:  "my super awesome desc",
	}
	s.Flags.BoolVar(&lsCmd.long, "l", false, "long declaration")
	return s
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
