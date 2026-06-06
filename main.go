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

type tmuxSessionListCommandFunc func() ([]byte, error)

func tmuxSessonListCommand() ([]byte, error) {
	return exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
}

func getTmuxSessions(tmuxSessionListCommand tmuxSessionListCommandFunc) map[string]struct{} {
	out, err := tmuxSessionListCommand()
	if err != nil {
		return nil
	}

	names := make(map[string]struct{})
	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		if line != "" {
			names[line] = struct{}{}
		}
	}

	return names
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

func tmuxSwitch(sessionName string) {
	cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}

func tmuxNewSession(sessionName, sessionPath string) {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", sessionPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}

type Config struct {
	projectDirectories []string
}

func main() {
	config := Config{
		projectDirectories: []string{os.Getenv("HOME") + "/Documents/repos"},
	}

	flag.Parse()

	tmuxSessions := getTmuxSessions(tmuxSessonListCommand)

	dirs, err := gitDirs(config.projectDirectories[0], os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	if flag.Arg(0) == "connect" && flag.Arg(1) != "" {
		name := flag.Arg(1)

		if _, ok := tmuxSessions[name]; ok {
			tmuxSwitch(name)
		}

		path, ok := dirs[name]

		if ok {
			tmuxNewSession(name, path)
			tmuxSwitch(name)
		}
		return
	}

	scores := zoxideScores()

	type repo struct {
		name   string
		path   string
		score  float64
		active bool
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
