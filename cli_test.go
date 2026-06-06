package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tmux-sessions-build-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binPath = filepath.Join(dir, "tmux-sessions")
	out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("build failed: %v\n%s", err, out))
	}

	os.Exit(m.Run())
}

func TestCLIListActiveSessionsFirst(t *testing.T) {
	env := setupCLIEnv(t, []string{"alpha"})

	out := runCLI(t, env, nil)
	assertLines(t, out, []string{" alpha", " beta", " gamma"})
}

func TestCLIOpenRepo(t *testing.T) {
	env := setupCLIEnv(t, nil)

	out := runCLI(t, env, []string{"open", "beta"})
	if got := strings.TrimSpace(out); got != filepath.Join(env.reposRoot, "beta") {
		t.Fatalf("open beta = %q, want %q", got, filepath.Join(env.reposRoot, "beta"))
	}
}

func TestCLIOpenMissingRepo(t *testing.T) {
	env := setupCLIEnv(t, nil)

	stderr, exit := runCLIExpectError(t, env, []string{"open", "missing"})
	if exit == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr, "repo not found: missing") {
		t.Fatalf("unexpected error: %q", stderr)
	}
}

type cliEnv struct {
	home      string
	reposRoot string
	bin       string
}

func setupCLIEnv(t *testing.T, sessions []string) cliEnv {
	t.Helper()

	home := t.TempDir()
	reposRoot := filepath.Join(home, "Documents", "repos")
	if err := os.MkdirAll(reposRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"alpha", "beta", "gamma"} {
		initGitRepo(t, filepath.Join(reposRoot, name))
	}

	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(fakeTmuxScript(sessions)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "zoxide"), []byte(fakeZoxideScript()), 0o755); err != nil {
		t.Fatal(err)
	}

	return cliEnv{home: home, reposRoot: reposRoot, bin: bin}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func fakeTmuxScript(sessions []string) string {
	body := `#!/bin/sh
if [ "$1" = "list-sessions" ] && [ "$3" = "#{session_name}" ]; then
`
	for _, session := range sessions {
		body += fmt.Sprintf("  echo %q\n", session)
	}
	body += `fi
exit 0
`
	return body
}

func fakeZoxideScript() string {
	return `#!/bin/sh
exit 1
`
}

func (env cliEnv) cmdEnv() []string {
	return []string{
		"HOME=" + env.home,
		"PATH=" + env.bin + ":" + os.Getenv("PATH"),
	}
}

func runCLI(t *testing.T, env cliEnv, args []string) string {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env.cmdEnv()...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v", binPath, args, err)
	}
	return stdout.String()
}

func runCLIExpectError(t *testing.T, env cliEnv, args []string) (string, int) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env.cmdEnv()...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stderr.String(), 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s %v failed unexpectedly: %v", binPath, args, err)
	}
	return stderr.String(), exitErr.ExitCode()
}

func assertLines(t *testing.T, out string, want []string) {
	t.Helper()

	got := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(got) == 1 && got[0] == "" {
		got = nil
	}
	if len(got) != len(want) {
		t.Fatalf("output lines = %v, want %v\nraw:\n%s", got, want, out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}
