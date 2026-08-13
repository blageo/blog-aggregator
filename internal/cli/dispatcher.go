package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/blog-aggregator/internal/config"
	"example.com/blog-aggregator/internal/database"
	"github.com/google/uuid"
)

// State holds the shared dependencies (database access and config) used by
// command handlers.
type State struct {
	db      *database.Queries
	cfg_ptr *config.Config
}

// NewState builds a State from a database connection and a loaded config.
func NewState(db *database.Queries, cfg *config.Config) *State {
	return &State{db: db, cfg_ptr: cfg}
}

// Command represents a single invocation of a CLI command, i.e. its name and
// the arguments passed to it.
type Command struct {
	Name string
	Args []string
}

// Commands maps command names to their handler functions.
type Commands struct {
	handlers map[string]func(*State, Command) error // map of command names to their handler functions
}

// newTimestampedID generates a new UUID and matching CreatedAt/UpdatedAt
// timestamps for use in database create params.
func newTimestampedID() (uuid.UUID, time.Time, time.Time) {
	now := time.Now()
	return uuid.New(), now, now
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.db.GetUserByName(context.Background(), s.cfg_ptr.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		return handler(s, cmd, user)
	}
}

func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.Name]
	if !exists {
		return errors.New("command not found")
	}
	return handler(s, cmd)
}

func (c *Commands) RegisterCommand(name string, f func(*State, Command) error) {
	if c.handlers == nil {
		c.handlers = make(map[string]func(*State, Command) error)
	}
	c.handlers[name] = f
}
