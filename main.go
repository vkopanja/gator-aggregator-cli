package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	cfg "gator/internal/config"
	"gator/internal/core"
	"gator/internal/database"
	"gator/internal/handler"
	"gator/internal/rss"
	"os"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type commands struct {
	commands map[string]func(*core.State, core.Command) error
}

func main() {
	config, err := cfg.Read()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("postgres", config.DbUrl)
	if err != nil {
		panic(err)
	}
	dbQueries := database.New(db)
	currentState := &core.State{Config: config, Db: dbQueries}

	commands := commands{commands: make(map[string]func(*core.State, core.Command) error)}
	commands.register("help", func(s *core.State, _ core.Command) error {
		fmt.Println("Available commands:")
		for cmd := range commands.commands {
			fmt.Printf("  %s\n", cmd)
		}
		return nil
	})
	commands.register("login", handler.Login)
	commands.register("register", handler.Register)
	commands.register("reset", handler.Reset)
	commands.register("users", handler.GetUsers)
	commands.register("addfeed", middlewareLoggedIn(handler.AddFeed))
	commands.register("feeds", handler.FetchFeeds)
	commands.register("follow", middlewareLoggedIn(handler.FollowFeed))
	commands.register("following", middlewareLoggedIn(handler.FeedFollowsForUser))
	commands.register("unfollow", middlewareLoggedIn(handler.UnfollowFeed))
	commands.register("agg", handlerAgg)
	commands.register("browse", middlewareLoggedIn(handler.Browse))

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: gator <core.Command> [args]")
		os.Exit(1)
	}

	if err := commands.run(currentState, core.Command{Name: args[1], Args: args[2:]}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlerAgg(s *core.State, cmd core.Command) error {
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

func middlewareLoggedIn(handler func(s *core.State, cmd core.Command, user database.User) error) func(*core.State, core.Command) error {
	return func(s *core.State, cmd core.Command) error {
		user, err := s.Db.GetUser(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		return handler(s, cmd, user)
	}
}

func (c *commands) run(s *core.State, cmd core.Command) error {
	cmdHandler, ok := c.commands[cmd.Name]
	if !ok {
		return fmt.Errorf("core.Command %s not found", cmd.Name)
	}

	if err := cmdHandler(s, cmd); err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*core.State, core.Command) error) {
	c.commands[name] = f
}
