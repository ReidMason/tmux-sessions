package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/ReidMason/tmux-sessions/config"
	"github.com/ReidMason/tmux-sessions/gitdirs"
	"github.com/ReidMason/tmux-sessions/tmux"
	"github.com/ReidMason/tmux-sessions/zoxide"
)

type TmuxHandler interface {
	GetSessions() map[string]struct{}
	CapturePane(sessionName string) (string, error)
	Switch(sessionName, sessionPath string) error
}

type ZoxideHandler interface {
	GetScores() map[string]float64
}

func handleConnectCommand(name string, roots []string, tmuxHandler TmuxHandler) {
	if name == "" {
		fmt.Println("Please provide a session name")
	}

	tmuxSessions := tmuxHandler.GetSessions()
	if _, ok := tmuxSessions[name]; ok {
		tmuxHandler.Switch(name, "")
		return
	}

	dirs, err := gitdirs.ProjectDirs(roots, os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		return
	}

	path, ok := dirs[name]
	if ok {
		tmuxHandler.Switch(name, path)
	}
}

type listItem struct {
	name  string
	score float64
	icon  string
}

func zoxideScoreForSession(name string, dirs map[string]string, scores map[string]float64) float64 {
	path, ok := dirs[name]
	if !ok {
		return 0
	}
	return scores[path]
}

func sortByScoreThenName(items []listItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].name < items[j].name
	})
}

func listDirectory(path string) (string, error) {
	if _, err := exec.LookPath("eza"); err == nil {
		out, err := exec.Command("eza", "--all", "--git", "--icons", "--color=always", path).CombinedOutput()
		return string(out), err
	}
	out, err := exec.Command("ls", "-la", path).CombinedOutput()
	return string(out), err
}

func handlePreviewCommand(name string, roots []string, tmuxHandler TmuxHandler) {
	if name == "" {
		fmt.Fprintln(os.Stderr, "please provide a session or directory name")
		os.Exit(1)
	}

	tmuxSessions := tmuxHandler.GetSessions()
	if tmuxSessions != nil {
		if _, ok := tmuxSessions[name]; ok {
			out, err := tmuxHandler.CapturePane(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error capturing pane: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(out)
			return
		}
	}

	dirs, err := gitdirs.ProjectDirs(roots, os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	path, ok := dirs[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "not found: %s\n", name)
		os.Exit(1)
	}

	out, err := listDirectory(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(out)
}

func handleSessionListCommand(roots []string, tmuxHandler TmuxHandler, zoxideHandler ZoxideHandler) {
	dirs, err := gitdirs.ProjectDirs(roots, os.ReadDir, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	scores := zoxideHandler.GetScores()
	tmuxSessions := tmuxHandler.GetSessions()

	sessions := make([]listItem, 0, len(tmuxSessions))
	for name := range tmuxSessions {
		sessions = append(sessions, listItem{
			name:  name,
			score: zoxideScoreForSession(name, dirs, scores),
			icon:  "\uf489",
		})
	}
	sortByScoreThenName(sessions)

	directories := make([]listItem, 0, len(dirs))
	for name, path := range dirs {
		if tmuxSessions != nil {
			if _, hasSession := tmuxSessions[name]; hasSession {
				continue
			}
		}
		directories = append(directories, listItem{
			name:  name,
			score: scores[path],
			icon:  "\uf07b",
		})
	}
	sortByScoreThenName(directories)

	for _, item := range sessions {
		fmt.Printf("%s %s\n", item.icon, item.name)
	}
	for _, item := range directories {
		fmt.Printf("%s %s\n", item.icon, item.name)
	}
}

func main() {
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	command := flag.Arg(0)
	switch command {
	case "list":
		tmuxHandler := tmux.New()
		zoxideHandler := zoxide.New()
		handleSessionListCommand(cfg.ProjectDirectories, tmuxHandler, zoxideHandler)
	case "connect":
		tmuxHandler := tmux.New()
		handleConnectCommand(flag.Arg(1), cfg.ProjectDirectories, tmuxHandler)
	case "preview":
		tmuxHandler := tmux.New()
		handlePreviewCommand(flag.Arg(1), cfg.ProjectDirectories, tmuxHandler)
	}
}
