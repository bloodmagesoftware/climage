//go:build linux || freebsd || openbsd || netbsd || dragonfly
// +build linux freebsd openbsd netbsd dragonfly

package downloads

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Try XDG user dirs file
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if cfgHome == "" {
		cfgHome = filepath.Join(home, ".config")
	}
	f := filepath.Join(cfgHome, "user-dirs.dirs")
	if path, err := parseXDGUserDirs(f, home); err == nil && path != "" {
		return path, nil
	}

	// xdg-user-dir command if present (best-effort)
	if path, err := tryXdgUserDirCommand(); err == nil && path != "" {
		return path, nil
	}

	// Fallback to ~/Downloads
	return filepath.Join(home, "Downloads"), nil
}

func parseXDGUserDirs(fpath, home string) (string, error) {
	fd, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// expecting: XDG_DOWNLOAD_DIR="$HOME/Downloads"
		if !strings.HasPrefix(line, "XDG_DOWNLOAD_DIR") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")
		// Expand $HOME and ~
		val = strings.ReplaceAll(val, "$HOME", home)
		if strings.HasPrefix(val, "~/") {
			val = filepath.Join(home, val[2:])
		}
		if val == "" {
			continue
		}
		return filepath.Clean(val), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no XDG_DOWNLOAD_DIR found")
}

func tryXdgUserDirCommand() (string, error) {
	// Best-effort: try to run `xdg-user-dir DOWNLOAD` if available.
	// Keep it optional — if command isn't present, ignore errors.
	xdg := "/usr/bin/xdg-user-dir"
	if _, err := os.Stat(xdg); err != nil {
		// try PATH lookup
		xdg = "xdg-user-dir"
	}
	out, err := runCommandCapture(xdg, "DOWNLOAD")
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "" || strings.HasPrefix(out, "XDG") {
		return "", errors.New("xdg-user-dir returned empty")
	}
	// Expand $HOME if present
	home, _ := os.UserHomeDir()
	out = strings.ReplaceAll(out, "$HOME", home)
	return out, nil
}

func runCommandCapture(name string, args ...string) (string, error) {
	// isolated import to avoid pulling os/exec on unsupported platforms
	// but this file is unix-only so it's okay to use os/exec
	// simple wrapper to run command and capture stdout
	cmd := []string{name}
	cmd = append(cmd, args...)
	c := execCommand(cmd[0], cmd[1:]...)
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
