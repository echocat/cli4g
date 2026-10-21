package cli

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed completion_powershell.ps1
var powershellCompletionScript string

func quotePowerShell(value string) string {
	return "'" + strings.NewReplacer("'", "''", "\u2018", "\u2018\u2018", "\u2019", "\u2019\u2019", "\u201a", "\u201a\u201a", "\u201b", "\u201b\u201b").Replace(value) + "'"
}

func powershellCompletionProfiles(ctx context.Context) ([]string, error) {
	var profiles []string
	seen := make(map[string]bool)
	for _, name := range []string{"pwsh", "powershell"} {
		shell, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		output, err := exec.CommandContext(ctx, shell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); [Console]::Write($PROFILE.CurrentUserAllHosts)").Output()
		if err != nil {
			return nil, fmt.Errorf("could not discover %s user profile: %w", name, err)
		}
		path := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("%s returned no absolute user profile path", name)
		}
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if !seen[key] {
			profiles = append(profiles, path)
			seen[key] = true
		}
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("could not discover PowerShell user profiles: install pwsh or powershell, or supply an installation directory")
	}
	return profiles, nil
}

func installPowerShellProfile(ctx context.Context, profile, script, command string) error {
	hash := sha256.Sum256([]byte(command))
	marker := fmt.Sprintf("# cli4g completion %x", hash[:8])
	// ASCII source preserves legacy ANSI profiles too. The path is data, never
	// interpolated PowerShell code, including when it contains quotes or Unicode.
	block := "\r\n" + marker + "\r\n. ([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String('" +
		base64.StdEncoding.EncodeToString([]byte(script)) + "')))\r\n# end cli4g completion\r\n"
	return appendCompletionProfile(ctx, profile, marker, block)
}
