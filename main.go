package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ReidMason/tmux-sessions/tmux"
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

// zoxideScores returns a map of absolute path → frecency score from zoxide.
// Returns an empty map if zoxide is unavailable or fails.
func zoxideScores() map[string]float64 {
	out, err := exec.Command("zoxide", "query", "--list", "--score").Output()
	scores := make(map[string]float64)
	if err != nil {
		return scores
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		score, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			continue
		}
		scores[parts[1]] = score
	}
	return scores
}

type Config struct {
	projectDirectories []string
}

type TmuxHandler interface {
	GetSessions() map[string]struct{}
	Switch(sessionName, sessionPath string) error
}

func handleConnectCommand(name string, tmuxSessions map[string]struct{}, config Config, tmuxHandler *tmux.Tmux) {
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

func main() {
	config := Config{
		projectDirectories: []string{os.Getenv("HOME") + "/Documents/repos"},
	}

	flag.Parse()

	tmuxHandler := tmux.New()

	tmuxSessions := tmuxHandler.GetSessions()

	if flag.Arg(0) == "connect" && flag.Arg(1) != "" {
		name := flag.Arg(1)
		handleConnectCommand(name, tmuxSessions, config, tmuxHandler)
		return
	}

	scores := zoxideScores()

	type repo struct {
		name   string
		path   string
		score  float64
		active bool
	}

	dirs, err := gitDirs(config.projectDirectories[0], os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

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
		fmt.Println(" " + r.name)
	}
}
