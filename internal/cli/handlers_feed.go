package cli

import (
	"errors"
	"fmt"
	"strings"

	"example.com/blog-aggregator/internal/database"
)

// HandlerAggregateFeeds fetches a feed and prints its contents to the
// terminal.
func HandlerAggregateFeeds(s *State, cmd Command) error {
	feedURL := "https://www.wagslane.dev/index.xml"

	feed, err := fetchFeed(s.ctx, feedURL)
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

// HandlerAddFeed creates a new feed owned by the logged-in user and makes
// them follow it.
func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return errors.New("not enough arguments provided for addfeed command, usage: addfeed <name> <url>")
	}
	name := cmd.Args[0]
	url := cmd.Args[1]

	feedID, feedCreatedAt, feedUpdatedAt := newTimestampedID()
	feedParams := database.CreateFeedParams{
		ID:        feedID,
		CreatedAt: feedCreatedAt,
		UpdatedAt: feedUpdatedAt,
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}

	feed, err := s.db.CreateFeed(s.ctx, feedParams)
	if err != nil {
		return fmt.Errorf("failed to create feed in database: %w", err)
	}

	followID, followCreatedAt, followUpdatedAt := newTimestampedID()
	_, err = s.db.CreateFeedFollow(s.ctx, database.CreateFeedFollowParams{
		ID:        followID,
		CreatedAt: followCreatedAt,
		UpdatedAt: followUpdatedAt,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow in database: %w", err)
	}

	fmt.Printf("Feed '%s' added successfully for user '%s'.\n", name, user.Name)
	fmt.Printf("Feed record: %+v\n", feed)

	return nil
}

// HandlerGetFeeds prints all feeds registered by any user.
func HandlerGetFeeds(s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return errors.New("feeds command does not take any arguments")
	}

	feeds, err := s.db.GetFeeds(s.ctx)
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

// HandlerFollowFeed makes the logged-in user follow an existing feed by URL.
func HandlerFollowFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("no feed URL provided for follow command")
	}

	feedURL := cmd.Args[0]

	feed, err := s.db.GetFeedByURL(s.ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to get feed by URL: %w", err)
	}

	id, createdAt, updatedAt := newTimestampedID()
	followParams := database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		UserID:    user.ID,
		FeedID:    feed.FeedID,
	}

	followRow, err := s.db.CreateFeedFollow(s.ctx, followParams)
	if err != nil {
		return fmt.Errorf("failed to create feed follow in database: %w", err)
	}

	fmt.Printf("User '%s' is now following feed '%s'.\n", followRow.UserName, followRow.FeedName)

	return nil
}

// HandlerUnfollowFeed makes the logged-in user unfollow a feed by URL.
func HandlerUnfollowFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("no feed URL provided for unfollow command")
	}

	feedURL := cmd.Args[0]

	feed, err := s.db.GetFeedByURL(s.ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to get feed by URL: %w", err)
	}

	err = s.db.DeleteFeedFollow(s.ctx, database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.FeedID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete feed follow: %w", err)
	}

	fmt.Printf("User '%s' is no longer following feed '%s'.\n", user.Name, feed.FeedName)

	return nil
}

// HandlerPrintFeedsForUser prints all feeds the logged-in user follows.
func HandlerPrintFeedsForUser(s *State, cmd Command, user database.User) error {
	feeds, err := s.db.GetFeedFollowsForUser(s.ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("- %s by %s\n", feed.FeedName, feed.UserName)
	}

	return nil
}
