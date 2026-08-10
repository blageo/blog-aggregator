package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
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

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
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

func handlerGetUsers(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return errors.New("users command does not take any arguments")
	}
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.Name == s.cfg_ptr.CurrentUserName {
			fmt.Println(user.Name, "(current)")
		} else {
			fmt.Println(user.Name)
		}
	}
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

func handlerAggregateFeeds(s *state, cmd command) error {
	/* uncomment to allow user passed feeds in CLI
	if len(cmd.args) < 1 {
		return errors.New("no feed URL provided for aggregation")
	}
	feedURL := cmd.args[0]
	*/
	feedURL := "https://www.wagslane.dev/index.xml"

	feed, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Feed Title: %s\n", feed.Channel.Title)
	fmt.Printf("Description: %s\n", feed.Channel.Description)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Items (%d):\n\n", len(feed.Channel.Item))
	for _, item := range feed.Channel.Item {
		fmt.Printf("• %s (%s)\n", item.Title, item.Link)
		fmt.Printf("  Published - %s\n", item.PubDate)
		if item.Description != "" {
			fmt.Println()
			fmt.Printf("  Content - %s\n", item.Description)
		}
		fmt.Println()
	}
	fmt.Println(strings.Repeat("=", 60))

	return nil
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("not enough arguments provided for addfeed command, usage: addfeed <name> <url>")
	}
	name := cmd.args[0]
	url := cmd.args[1]

	user, err := s.db.GetUserByName(context.Background(), s.cfg_ptr.CurrentUserName)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}

	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("failed to create feed in database: %w", err)
	}

	fmt.Printf("Feed '%s' added successfully for user '%s'.\n", name, user.Name)
	fmt.Printf("Feed record: %+v\n", feed)

	return nil
}

func handlerGetFeeds(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return errors.New("feeds command does not take any arguments")
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("- %s (%s) by %s\n", feed.FeedName, feed.Url, feed.UserName)
	}

	return nil

}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch feed: %s", resp.Status)
	}

	var feed RSSFeed
	err = xml.NewDecoder(resp.Body).Decode(&feed)
	if err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil

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
