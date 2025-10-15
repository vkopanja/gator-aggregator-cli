package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	cfg "gator/internal/config"
	"gator/internal/database"
	"gator/internal/rss"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type state struct {
	db     *database.Queries
	config *cfg.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	commands map[string]func(*state, command) error
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
	currentState := &state{config: config, db: dbQueries}

	commands := commands{commands: make(map[string]func(*state, command) error)}
	commands.register("help", func(s *state, _ command) error {
		fmt.Println("Available commands:")
		for cmd := range commands.commands {
			fmt.Printf("  %s\n", cmd)
		}
		return nil
	})
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerGetUsers)
	commands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	commands.register("feeds", handlerFetchFeeds)
	commands.register("follow", middlewareLoggedIn(handlerFollowFeed))
	commands.register("following", middlewareLoggedIn(handlerFeedFollowsForUser))
	commands.register("unfollow", middlewareLoggedIn(handlerUnfollowFeed))
	commands.register("agg", handlerAgg)
	commands.register("browse", middlewareLoggedIn(handlerBrowse))

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: gator <command> [args]")
		os.Exit(1)
	}

	if err := commands.run(currentState, command{name: args[1], args: args[2:]}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the login handler expects a single argument, the username")
	}

	user, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("the user with that name does not exist")
	}

	if err := s.config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed setting user: %s", err)
	}

	fmt.Printf("User successfully set to %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the register handler expects a single argument, the username")
	}

	user, _ := s.db.GetUser(context.Background(), cmd.args[0])
	if user.ID != uuid.Nil {
		fmt.Println("user with that name already exists")
		os.Exit(1)
	}
	createUser, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		Name:      cmd.args[0],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return err
	} else {
		fmt.Printf("User %s created\n", createUser.Name)
		err := s.config.SetUser(createUser.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func handlerReset(s *state, _ command) error {
	err := s.db.ClearUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed clearing users: %s", err)
	} else {
		fmt.Println("users cleared")
	}
	return nil
}

func handlerGetUsers(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed getting users: %s", err)
	}
	for _, user := range users {
		current := s.config.CurrentUserName == user.Name
		if current {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("the addfeed handler expects a two params, name and url")
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		UserID:    currentUser.ID,
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed creating feed: %s\n", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
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

func handlerFetchFeeds(s *state, _ command) error {
	feeds, err := s.db.GetFeedsWithUserName(context.Background())
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

func handlerFollowFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("the follow handler expects a single argument, the feed url")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("feed with the requested url does not exist")
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
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

func handlerFeedFollowsForUser(s *state, _ command, currentUser database.User) error {
	feeds, err := s.db.GetFeedsForUser(context.Background(), currentUser.Name)
	if err != nil {
		return fmt.Errorf("failed getting feeds for user: %s\n", err)
	}

	for _, feed := range feeds {
		fmt.Printf("- '%s'\n", feed.Name)
	}

	return nil
}

func handlerUnfollowFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("the unfollow handler expects a single argument, the feed url")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("feed with the requested url does not exist")
	}

	_, err = s.db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed deleting feed follow: %s\n", err)
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("the agg handler expects a single argument, the time interval how oftern to fetch feeds")
	}

	duration, err := time.ParseDuration(cmd.args[0])
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

func handlerBrowse(s *state, cmd command, currentUser database.User) error {
	var limit int
	if len(cmd.args) < 1 {
		limit = 2
	} else {
		parsedInt, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			limit = 2
		} else {
			limit = parsedInt
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
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

func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("failed getting next feed to fetch: %s", err)
	}

	fmt.Printf("Fetching items for feed: %s\n", nextFeed.Name)

	_, err = s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
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
		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		return handler(s, cmd, user)
	}
}

func (c *commands) run(s *state, cmd command) error {
	cmdHandler, ok := c.commands[cmd.name]
	if !ok {
		return fmt.Errorf("command %s not found", cmd.name)
	}

	if err := cmdHandler(s, cmd); err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commands[name] = f
}
