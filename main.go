package main

import (
	"context"
	"database/sql"
	"fmt"
	cfg "gator/internal/config"
	"gator/internal/database"
	"os"
	"time"

	"github.com/google/uuid"
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
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerGetUsers)

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
