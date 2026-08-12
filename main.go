package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"example.com/blog-aggregator/internal/config"
	"example.com/blog-aggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	s := state{cfg_ptr: &cfg}

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	s.db = dbQueries

	cmds := commands{
		handlers: make(map[string]func(*state, command) error),
	}
	cmds.registerCommand("login", handlerLogin)
	cmds.registerCommand("register", handlerRegisterUser)
	cmds.registerCommand("reset", handlerReset)
	cmds.registerCommand("users", handlerGetUsers)
	cmds.registerCommand("agg", handlerAggregateFeeds)
	cmds.registerCommand("addfeed", handlerAddFeed)
	cmds.registerCommand("feeds", handlerGetFeeds)
	cmds.registerCommand("follow", handlerFollowFeed)
	cmds.registerCommand("following", handlerPrintFeedsForUser)

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
