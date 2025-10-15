package main

import (
	"context"
	"database/sql"
	"fmt"
	cfg "gator/internal/config"
	"gator/internal/core"
	"gator/internal/database"
	"gator/internal/handler"
	"os"

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
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Printf("failed closing database connection: %s\n", err)
			os.Exit(1)
		}
	}(db)

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
	commands.register("agg", handler.AggregateFeeds)
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
