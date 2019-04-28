package cli_test

import (
	"context"
	"golang.org/x/xerrors"
	"log"
	"nhooyr.io/cli"
	"os"
	"os/exec"
	"time"
)

type rootCmd struct {
	status int
}

func (rootCmd *rootCmd) Spec() cli.Command {
	s := cli.Command{
		Name:  os.Args[0],
		Usage: "[-fail] <SUBCMD>",
		Desc:  "my awesome description.",
		Subcommands: []cli.Handler{
			&lsCmd{
				rootCmd: rootCmd,
			},
		},
	}
	s.Flags.IntVar(&rootCmd.status, "fail", 0, "exit with given status")
	return s
}

func (rootCmd *rootCmd) Run(args []string) int {
	return rootCmd.status
}

type lsCmd struct {
	rootCmd *rootCmd
	long    bool
}

func (lsCmd *lsCmd) Spec() cli.Command {
	s := cli.Command{
		Name:  "ls",
		Usage: "<DIR>",
		Desc:  "my super awesome desc",
	}
	s.Flags.BoolVar(&lsCmd.long, "l", false, "long declaration")
	return s
}

func (lsCmd *lsCmd) Run(args []string) int {
	if len(args) != 1 {
		log.Println("you must provide a single argument")
		return cli.Help()
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
		var cerr *exec.ExitError
		if !xerrors.As(err, &cerr) {
			log.Println("failed to wait for %q: %v", ls.Args, err)
			return 1
		}
	}

	return 0
}

func Example() {
	log.SetFlags(0)
	cli.Run(&rootCmd{})
}
