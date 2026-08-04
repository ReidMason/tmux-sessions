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
	return exec.Command("tmux", "has-session", "-t", sessionTarget(sessionName)).Run() == nil
}

func (t Tmux) CapturePane(sessionName string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", sessionTarget(sessionName), "-p").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (t Tmux) Switch(sessionName, sessionPath string) error {
	inTmux := os.Getenv("TMUX") != ""
	if t.SessionExists(sessionName) {
		if inTmux {
			switchToExistingSession(sessionName)
			return nil
		}
		attachToExistingSession(sessionName)
		return nil
	}

	if sessionPath == "" {
		return fmt.Errorf("session %q not found", sessionName)
	}

	if inTmux {
		return createAndSwitchToNewSession(sessionName, sessionPath)
	}

	return createNewSession(sessionName, sessionPath)
}

func createNewSession(sessionName, sessionPath string) error {
	cmd := exec.Command("tmux", "new-session", "-s", sessionName, "-c", sessionPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createNewBackgroundSession(sessionName, sessionPath string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", sessionPath).Run()
}

func createAndSwitchToNewSession(sessionName, sessionPath string) error {
	if err := createNewBackgroundSession(sessionName, sessionPath); err != nil {
		return err
	}

	return switchToExistingSession(sessionName)
}

func switchToExistingSession(sessionName string) error {
	return exec.Command("tmux", "switch-client", "-t", sessionTarget(sessionName)).Run()
}

func attachToExistingSession(sessionName string) {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionTarget(sessionName))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// sessionTarget forces an exact-match target so session names containing
// tmux's special "." (window) or ":" (window) separators are treated literally.
func sessionTarget(sessionName string) string {
	return "=" + sessionName
}
