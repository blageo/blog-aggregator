package main

import (
	"fmt"
	"log"
	"os"

	"example.com/blog-aggregator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	s := state{cfg_ptr: &cfg}

	cmds := commands{
		handlers: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments provided, a command name is required")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]
	cmd := command{name: cmdName, args: cmdArgs}

	err = cmds.run(&s, cmd)
	if err != nil {
		log.Fatal(err)
	}
}
