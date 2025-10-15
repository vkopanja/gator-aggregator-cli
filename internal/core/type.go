package core

import (
	cfg "gator/internal/config"
	"gator/internal/database"
)

type State struct {
	Db     *database.Queries
	Config *cfg.Config
}

type Command struct {
	Name string
	Args []string
}
