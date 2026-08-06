package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"example.com/blog-aggregator/internal/config"
	"example.com/blog-aggregator/internal/database"
	"github.com/google/uuid"
)

type state struct {
	db      *database.Queries
	cfg_ptr *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error // map of command names to their handler functions
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return errors.New("no arguments provided for login command")
	}
	if len(cmd.args) > 1 {
		return errors.New("usage: login <username>")
	}
	if _, err := s.db.GetUserByName(context.Background(), cmd.args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "user does not exist:", err)
		os.Exit(1)
	}
	err := s.cfg_ptr.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	println("User set to:", cmd.args[0])
	return nil
}

func handlerRegisterUser(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return errors.New("no arguments provided for register command")
	}

	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	}

	_, err := s.db.CreateUser(context.Background(), userParams)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create user:", err)
		os.Exit(1)
	}

	err = s.cfg_ptr.SetUser(cmd.args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not set current user:", err)
		os.Exit(1)
	}
	println("User registered:", cmd.args[0])
	userData, err := s.db.GetUserByName(context.Background(), cmd.args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not fetch newly created user:", err)
		os.Exit(1)
	}
	fmt.Printf("User data: %+v\n", userData)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return errors.New("reset command does not take any arguments")
	}
	err := s.db.Reset(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not reset database:", err)
		os.Exit(1)
	}
	println("Database reset successfully.")
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return errors.New("command not found")
	}
	return handler(s, cmd)
}

func (c *commands) registerCommand(name string, f func(*state, command) error) {
	if c.handlers == nil {
		c.handlers = make(map[string]func(*state, command) error)
	}
	c.handlers[name] = f
}
