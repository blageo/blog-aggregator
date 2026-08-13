package cli

import (
	"errors"
	"fmt"

	"example.com/blog-aggregator/internal/database"
)

// HandlerLogin sets the current user in the config, provided the user
// already exists in the database.
func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("no arguments provided for login command")
	}
	if len(cmd.Args) > 1 {
		return errors.New("usage: login <username>")
	}
	if _, err := s.db.GetUserByName(s.ctx, cmd.Args[0]); err != nil {
		return fmt.Errorf("user does not exist: %w", err)
	}
	err := s.cfg.SetUser(cmd.Args[0])
	if err != nil {
		return err
	}
	fmt.Println("User set to:", cmd.Args[0])
	return nil
}

// HandlerRegisterUser creates a new user in the database and sets them as
// the current user in the config.
func HandlerRegisterUser(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("no arguments provided for register command")
	}

	id, createdAt, updatedAt := newTimestampedID()
	userParams := database.CreateUserParams{
		ID:        id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Name:      cmd.Args[0],
	}

	_, err := s.db.CreateUser(s.ctx, userParams)
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	err = s.cfg.SetUser(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("could not set current user: %w", err)
	}
	fmt.Println("User registered:", cmd.Args[0])
	userData, err := s.db.GetUserByName(s.ctx, cmd.Args[0])
	if err != nil {
		return fmt.Errorf("could not fetch newly created user: %w", err)
	}
	fmt.Printf("User data: %+v\n", userData)
	return nil
}

// HandlerGetUsers prints all registered users, marking the current one.
func HandlerGetUsers(s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return errors.New("users command does not take any arguments")
	}
	users, err := s.db.GetUsers(s.ctx)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Println(user.Name, "(current)")
		} else {
			fmt.Println(user.Name)
		}
	}
	return nil
}

// HandlerReset wipes the database.
func HandlerReset(s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return errors.New("reset command does not take any arguments")
	}
	err := s.db.Reset(s.ctx)
	if err != nil {
		return fmt.Errorf("could not reset database: %w", err)
	}
	fmt.Println("Database reset successfully.")
	return nil
}
