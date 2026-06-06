package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/ReidMason/tmux-sessions/tmux"
	"github.com/ReidMason/tmux-sessions/zoxide"
)

type readDirFunc func(name string) ([]fs.DirEntry, error)
type statDirFunc func(name string) (os.FileInfo, error)

func gitDirs(root string, readDir readDirFunc, statDir statDirFunc) (map[string]string, error) {
	entries, err := readDir(root)
	if err != nil {
		return nil, err
	}

	dirs := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := statDir(filepath.Join(path, ".git")); err == nil {
			dirs[entry.Name()] = path
		}
	}

	return dirs, nil
}

type Config struct {
	projectDirectories []string
}

type TmuxHandler interface {
	GetSessions() map[string]struct{}
	Switch(sessionName, sessionPath string) error
}

type ZoxideHandler interface {
	GetScores() map[string]float64
}

func handleConnectCommand(name string, config Config, tmuxHandler TmuxHandler) {
	if name == "" {
		fmt.Println("Please provide a session name")
	}

	tmuxSessions := tmuxHandler.GetSessions()
	if _, ok := tmuxSessions[name]; ok {
		tmuxHandler.Switch(name, "")
		return
	}

	dirs, err := gitDirs(config.projectDirectories[0], os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		return
	}

	path, ok := dirs[name]
	if ok {
		tmuxHandler.Switch(name, path)
	}
}

type repo struct {
	name   string
	path   string
	score  float64
	active bool
}

func handleSessionListCommand(config Config, tmuxHandler TmuxHandler, zoxideHandler ZoxideHandler) {
	dirs, err := gitDirs(config.projectDirectories[0], os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	scores := zoxideHandler.GetScores()
	tmuxSessions := tmuxHandler.GetSessions()

	repos := make([]repo, 0, len(dirs))
	for name, path := range dirs {
		_, isActive := tmuxSessions[name]
		repos = append(repos, repo{
			name:   name,
			path:   path,
			score:  scores[path],
			active: tmuxSessions != nil && isActive,
		})
	}

	sort.Slice(repos, func(i, j int) bool {
		if repos[i].active != repos[j].active {
			return repos[i].active
		}
		if repos[i].score != repos[j].score {
			return repos[i].score > repos[j].score
		}
		return repos[i].name < repos[j].name
	})

	for _, r := range repos {
		prefix := "\uf07b"
		if r.active {
			prefix = "\uf489"
		}
		fmt.Printf("%s %s\n", prefix, r.name)
	}
}

func main() {
	config := Config{
		projectDirectories: []string{os.Getenv("HOME") + "/Documents/repos"},
	}

	flag.Parse()
	// sesh list -t --icons

	command := flag.Arg(0)
	switch command {
	case "list":
		tmuxHandler := tmux.New()
		zoxideHandler := zoxide.New()
		handleSessionListCommand(config, tmuxHandler, zoxideHandler)
	case "connect":
		tmuxHandler := tmux.New()
		handleConnectCommand(flag.Arg(1), config, tmuxHandler)
	}
}
