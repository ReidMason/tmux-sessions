package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func gitDirs(root string) (map[string]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	dirs := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			dirs[entry.Name()] = path
		}
	}
	return dirs, nil
}

// tmuxSessionNames returns active tmux session names, or nil if tmux is unavailable.
func tmuxSessionNames() map[string]struct{} {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
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

func main() {
	all := flag.Bool("all", false, "include repos that already have a tmux session with the same name")
	flag.Parse()

	root := os.Getenv("HOME") + "/Documents/repos"

	dirs, err := gitDirs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	if flag.Arg(0) == "open" && flag.Arg(1) != "" {
		path, ok := dirs[flag.Arg(1)]
		if !ok {
			fmt.Fprintf(os.Stderr, "repo not found: %s\n", flag.Arg(1))
			os.Exit(1)
		}
		fmt.Println(path)
		return
	}

	scores := zoxideScores()
	active := tmuxSessionNames()

	type repo struct {
		name  string
		path  string
		score float64
	}
	repos := make([]repo, 0, len(dirs))
	for name, path := range dirs {
		if !*all && active != nil {
			if _, exists := active[name]; exists {
				continue
			}
		}
		repos = append(repos, repo{name: name, path: path, score: scores[path]})
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].score > repos[j].score
	})

	for _, r := range repos {
		fmt.Println(" " + r.name)
	}
}
