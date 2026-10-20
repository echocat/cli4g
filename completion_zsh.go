package cli

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed completion_zsh.zsh
var zshCompletionScript string

func zshCompletionInstallation(ctx context.Context) (directory, profile string, err error) {
	output, err := exec.CommandContext(ctx, "zsh", "-c", "print -rn -- ${ZDOTDIR:-$HOME}").Output()
	if err != nil {
		return "", "", fmt.Errorf("could not discover Zsh startup directory: %w", err)
	}
	startup := strings.TrimSpace(string(output))
	if !filepath.IsAbs(startup) || strings.ContainsAny(startup, "\r\n\x00") {
		return "", "", fmt.Errorf("zsh returned no absolute startup directory")
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		data = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(data, "zsh", "site-functions"), filepath.Join(startup, ".zshrc"), nil
}

func installZshProfile(ctx context.Context, profile, directory, command string) error {
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	hash := sha256.Sum256([]byte(command))
	marker := fmt.Sprintf("# cli4g completion %x", hash[:8])
	block := "\n" + marker + "\nfpath=(" + quote(directory) + " $fpath)\n" +
		"if (( ! $+functions[compdef] )); then\n  autoload -Uz compinit\n  compinit\nfi\n" +
		"autoload -Uz -- " + quote("_"+command) + "\ncompdef " + quote("_"+command) + " " + quote(command) +
		"\n# end cli4g completion\n"
	return appendCompletionProfile(ctx, profile, marker, block)
}
