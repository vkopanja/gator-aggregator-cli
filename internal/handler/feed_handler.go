package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/core"
	"gator/internal/database"
	"gator/internal/rss"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func AddFeed(s *core.State, cmd core.Command, currentUser database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("the addfeed handler expects a two params, name and url")
	}

	feed, err := s.Db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		UserID:    currentUser.ID,
		Name:      cmd.Args[0],
		Url:       cmd.Args[1],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed creating feed: %s\n", err)
	}

	_, err = s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		FeedID:    feed.ID,
		UserID:    currentUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed creating feed follow: %s\n", err)
	}

	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("User ID: %s\n", feed.UserID)

	return nil
}

func FollowFeed(s *core.State, cmd core.Command, currentUser database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("the follow handler expects a single argument, the feed url")
	}

	feed, err := s.Db.GetFeedByUrl(context.Background(), cmd.Args[0])
	if err != nil {
		return fmt.Errorf("feed with the requested url does not exist")
	}

	follow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		FeedID:    feed.ID,
		UserID:    currentUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed creating feed follow: %s\n", err)
	}

	fmt.Printf("Feed: %s\n", follow.FeedName)
	fmt.Printf("User: %s\n", follow.UserName)

	return nil
}

func FeedFollowsForUser(s *core.State, _ core.Command, currentUser database.User) error {
	feeds, err := s.Db.GetFeedsForUser(context.Background(), currentUser.Name)
	if err != nil {
		return fmt.Errorf("failed getting feeds for user: %s\n", err)
	}

	for _, feed := range feeds {
		fmt.Printf("- '%s'\n", feed.Name)
	}

	return nil
}

func UnfollowFeed(s *core.State, cmd core.Command, currentUser database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("the unfollow handler expects a single argument, the feed url")
	}

	feed, err := s.Db.GetFeedByUrl(context.Background(), cmd.Args[0])
	if err != nil {
		return fmt.Errorf("feed with the requested url does not exist")
	}

	_, err = s.Db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed deleting feed follow: %s\n", err)
	}

	return nil
}

func FetchFeeds(s *core.State, _ core.Command) error {
	feeds, err := s.Db.GetFeedsWithUserName(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("Name: %s\n", feed.Name)
		fmt.Printf("URL: %s\n", feed.Url)
		fmt.Printf("User: %s\n", feed.UserName)
		fmt.Println()
	}

	return nil
}

func Browse(s *core.State, cmd core.Command, currentUser database.User) error {
	var limit int
	if len(cmd.Args) < 1 {
		limit = 2
	} else {
		parsedInt, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			limit = 2
		} else {
			limit = parsedInt
		}
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: currentUser.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Printf("Title: %s\n", post.Title)
	}

	return nil
}

func AggregateFeeds(s *core.State, cmd core.Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("the agg handler expects a single argument, the time interval how oftern to fetch feeds")
	}

	duration, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("failed parsing duration: %s", err)
	}

	fmt.Printf("Collecting feeds every %s\n", duration.String()+"")

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Printf("failed scraping feed: %s\n", err)
		}
	}
}

func scrapeFeeds(s *core.State) error {
	nextFeed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("failed getting next feed to fetch: %s", err)
	}

	fmt.Printf("Fetching items for feed: %s\n", nextFeed.Name)

	_, err = s.Db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		return err
	}

	feed, err := rss.FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return fmt.Errorf("failed fetching feed: %s", err)
	}

	for _, item := range feed.Channel.Item {
		parsedTime, err := time.Parse(time.RFC1123Z, item.PubDate)
		validTime := err == nil
		if err != nil {
			fmt.Printf("failed parsing time for post %s: %s\n", item.Title, err)
		}
		_, err = s.Db.CreatePost(context.Background(), database.CreatePostParams{
			Title:       item.Title,
			Url:         sql.NullString{String: item.Link, Valid: item.Link != ""},
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: sql.NullTime{Time: parsedTime, Valid: validTime},
			FeedID:      nextFeed.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		if err != nil {
			var pqError *pq.Error
			if errors.As(err, &pqError) && pqError.Code == "23505" {
				// duplicate key for URL, we can either ignore it, updated it or just log it
				// for now we'll ignore
				continue
			}
			fmt.Printf("failed creating post with title %s: %s\n", item.Title, err)
		}
	}

	return nil
}
