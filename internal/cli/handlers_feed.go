package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/blageo/blog-aggregator/internal/database"
)

// ScrapeFeeds fetches the next feed to scrape from the database, marks it as fetched, and prints its items.
// meant to be run as a continuous loop inside handler for agg
func ScrapeFeeds(s *State) error {
	nextFeed, err := s.db.GetNextFeedToFetch(s.ctx)
	if err != nil {
		return fmt.Errorf("no feed to fetch: %w", err)
	}

	feedData, err := fetchFeed(s.ctx, nextFeed.Url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	if _, err := s.db.MarkFeedFetched(s.ctx, nextFeed.ID); err != nil {
		return fmt.Errorf("failed to mark feed as fetched: %w", err)
	}

	for _, item := range feedData.Channel.Item {
		var publishedAt sql.NullTime
		if parsed, err := parsePubDate(item.PubDate); err == nil {
			publishedAt = sql.NullTime{Time: parsed, Valid: true}
		}

		postID, postCreatedAt, postUpdatedAt := newTimestampedID()
		postParams := database.CreatePostParams{
			ID:          postID,
			CreatedAt:   postCreatedAt,
			UpdatedAt:   postUpdatedAt,
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: publishedAt,
			FeedID:      nextFeed.ID,
		}

		if _, err := s.db.CreatePost(s.ctx, postParams); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			log.Printf("failed to create post %q: %v", postParams.Url, err)
			continue
		}
	}

	return nil
}

// HandlerAgg is the main handler for the "agg" command, which continuously scrapes feeds in a loop.
func HandlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("must supply a duration between requests, usage: agg <duration> (e.g., 1s, 1m, 1h)")
	}

	durationStr := cmd.Args[0]
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	fmt.Printf("Collecting feeds every %s...\n", durationStr)

	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		err := ScrapeFeeds(s)
		if err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
		}
	}
}

// HandlerAggregateFeeds fetches a single hardcoded feed and prints its contents to the
// terminal.
func HandlerAggregateFeed(s *State, cmd Command) error {
	feedURL := "https://www.wagslane.dev/index.xml"

	feedData, err := fetchFeed(s.ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Feed Title: %s\n", feedData.Channel.Title)
	fmt.Printf("Description: %s\n", feedData.Channel.Description)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Items (%d):\n\n", len(feedData.Channel.Item))
	for _, item := range feedData.Channel.Item {
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

func HandlerBrowseUserFeeds(s *State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) > 1 {
		return errors.New("browse command takes at most one argument, usage: browse [limit]")
	}
	if len(cmd.Args) == 1 {
		parsedLimit, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit %q: %w", cmd.Args[0], err)
		}
		limit = parsedLimit
	}

	posts, err := s.db.GetPostsForUser(s.ctx, database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("failed to get posts for user: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	fmt.Println("Posts:")
	for _, post := range posts {
		fmt.Printf("- %s (%s)\n", post.Title, post.Url)
	}

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

	if !strings.Contains(url, "://") {
		return fmt.Errorf("invalid feed URL %q: usage: addfeed \"<name>\" <url>", url)
	}

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
