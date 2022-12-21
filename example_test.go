package cli_test

import (
	"flag"
)

type Meta struct {
	Name string
	Flags []Flag
}

type Command interface {
	Meta() Meta
	Run(ctx context.Context, args []string) int
}

func Example() {
	
}

type Flag interface {
	Name() string
	Unmarshal() error
}
