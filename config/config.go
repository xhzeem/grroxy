package config

import (
	"os"
	"path"
)

type Config struct {
	ProjectDirectory  string
	DatabaseDirectory string
	CacheDirectory    string
}

func (c *Config) NewProject() {
	os.MkdirAll(c.ProjectDirectory, 0644)
	os.Mkdir(path.Join(c.ProjectDirectory, "targets"), 0644)
	os.Mkdir(path.Join(c.ProjectDirectory, "repeater"), 0644)
	os.Mkdir(path.Join(c.ProjectDirectory, "intruder"), 0644)
}

func (c *Config) Setup() {
	c.ProjectDirectory = path.Join(c.DatabaseDirectory, c.ProjectDirectory)

	if _, err := os.Stat(c.ProjectDirectory); os.IsNotExist(err) {
		c.NewProject()
	}
}
