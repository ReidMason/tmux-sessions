package zoxide

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type Zoxide struct{}

func New() *Zoxide {
	return &Zoxide{}
}

func (z Zoxide) GetScores() map[string]float64 {
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
