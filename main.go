package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"example.com/blog-aggregator/internal/cli"
	"example.com/blog-aggregator/internal/config"
	"example.com/blog-aggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	s := cli.NewState(dbQueries, &cfg)

	cmds := cli.Commands{}
	cmds.RegisterCommand("login", cli.HandlerLogin)
	cmds.RegisterCommand("register", cli.HandlerRegisterUser)
	cmds.RegisterCommand("reset", cli.HandlerReset)
	cmds.RegisterCommand("users", cli.HandlerGetUsers)
	cmds.RegisterCommand("agg", cli.HandlerAggregateFeeds)
	cmds.RegisterCommand("addfeed", cli.MiddlewareLoggedIn(cli.HandlerAddFeed))
	cmds.RegisterCommand("feeds", cli.HandlerGetFeeds)
	cmds.RegisterCommand("follow", cli.MiddlewareLoggedIn(cli.HandlerFollowFeed))
	cmds.RegisterCommand("unfollow", cli.MiddlewareLoggedIn(cli.HandlerUnfollowFeed))
	cmds.RegisterCommand("following", cli.MiddlewareLoggedIn(cli.HandlerPrintFeedsForUser))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments provided, a command name is required")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]
	cmd := cli.Command{Name: cmdName, Args: cmdArgs}

	err = cmds.Run(s, cmd)
	if err != nil {
		log.Fatal(err)
	}
}
