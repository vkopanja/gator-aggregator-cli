package handler

import (
	"context"
	"fmt"
	"gator/internal/core"
	"gator/internal/database"
	"os"
	"time"

	"github.com/google/uuid"
)

func Login(s *core.State, cmd core.Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("the login handler expects a single argument, the username")
	}

	user, err := s.Db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		return fmt.Errorf("the user with that name does not exist")
	}

	if err := s.Config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed setting user: %s", err)
	}

	fmt.Printf("User successfully set to %s\n", cmd.Args[0])
	return nil
}

func Register(s *core.State, cmd core.Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("the register handler expects a single argument, the username")
	}

	user, _ := s.Db.GetUser(context.Background(), cmd.Args[0])
	if user.ID != uuid.Nil {
		fmt.Println("user with that name already exists")
		os.Exit(1)
	}
	createUser, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		Name:      cmd.Args[0],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return err
	} else {
		fmt.Printf("User %s created\n", createUser.Name)
		err := s.Config.SetUser(createUser.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetUsers(s *core.State, _ core.Command) error {
	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed getting users: %s", err)
	}
	for _, user := range users {
		current := s.Config.CurrentUserName == user.Name
		if current {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func Reset(s *core.State, _ core.Command) error {
	err := s.Db.ClearUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed clearing users: %s", err)
	} else {
		fmt.Println("users cleared")
	}
	return nil
}
