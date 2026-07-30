package main

import (
	"errors"

	"example.com/blog-aggregator/internal/config"
)

type state struct {
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
	} else if len(cmd.args) > 1 {
		return errors.New("usage: login <username>")
	}
	err := s.cfg_ptr.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	println("User set to:", cmd.args[0])
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return errors.New("command not found")
	}
	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	if c.handlers == nil {
		c.handlers = make(map[string]func(*state, command) error)
	}
	c.handlers[name] = f
}
