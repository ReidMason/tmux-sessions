package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Tmux struct{}

func New() *Tmux {
	return &Tmux{}
}

func (t Tmux) GetSessions() map[string]struct{} {
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

func (t Tmux) SessionExists(sessionName string) bool {
	return exec.Command("tmux", "has-session", "-t", sessionName).Run() == nil
}

func (t Tmux) Switch(sessionName, sessionPath string) error {
	inTmux := os.Getenv("TMUX") != ""
	if t.SessionExists(sessionName) {
		if inTmux {
			return exec.Command("tmux", "switch-client", "-t", sessionName).Run()
		}
		cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if sessionPath == "" {
		return fmt.Errorf("session %q not found", sessionName)
	}

	if inTmux {
		if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", sessionPath).Run(); err != nil {
			return err
		}
		return exec.Command("tmux", "switch-client", "-t", sessionName).Run()
	}

	cmd := exec.Command("tmux", "new-session", "-s", sessionName, "-c", sessionPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
