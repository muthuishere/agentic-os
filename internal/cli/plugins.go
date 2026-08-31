package cli

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// metadataScanLimit matches omarchy: only the first lines of a script are read
// looking for metadata, so discovery stays cheap over hundreds of files.
const metadataScanLimit = 80

// Prefixes are the file-name prefixes an external command may carry. The
// command is `aos`, so that is what a plugin author reaches for; the longer
// project name is accepted too, since both are obvious guesses.
var Prefixes = []string{"aos-", "agentic-os-"}

// Prefix is the canonical prefix, used in documentation and examples.
var Prefix = Prefixes[0]

// metaKey is the comment tag external commands use to describe themselves,
// e.g. `# agentic-os:summary=Apply a theme`.
const metaKey = "agentic-os:"

// PluginDirs lists, in order, where external commands are looked for:
// $AGENTIC_OS_BIN_DIR, the user config bin dir, then every PATH entry.
func PluginDirs(env func(string) string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" {
			return
		}
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	add(env("AGENTIC_OS_BIN_DIR"))
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".config", "agentic-os", "bin"))
	}
	for _, dir := range filepath.SplitList(env("PATH")) {
		add(dir)
	}
	return dirs
}

// DiscoverPlugins registers every external `agentic-os-<route>` executable it
// can find. Builtins already registered win: a duplicate route is skipped, so
// shipping a Go implementation of a command shadows a script of the same name.
func DiscoverPlugins(r *Registry, env func(string) string) {
	for _, dir := range PluginDirs(env) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			stem, ok := trimPluginPrefix(name)
			if !ok {
				continue
			}
			path := filepath.Join(dir, name)
			if !isExecutable(path) {
				continue
			}
			cmd := parsePlugin(path, stem)
			if cmd == nil {
				continue
			}
			if r.Lookup(cmd.Group, cmd.Name) != nil {
				continue
			}
			r.Add(cmd)
		}
	}
}

// trimPluginPrefix removes whichever accepted prefix a file carries.
func trimPluginPrefix(name string) (string, bool) {
	for _, prefix := range Prefixes {
		if strings.HasPrefix(name, prefix) {
			return strings.TrimPrefix(name, prefix), true
		}
	}
	return "", false
}

// parsePlugin turns a file name and its metadata header into a Command.
// `agentic-os-audio-output-volume` becomes the route `audio output volume`.
func parsePlugin(path, stem string) *Command {
	stem = strings.TrimSuffix(stem, filepath.Ext(stem))
	if stem == "" {
		return nil
	}
	tokens := strings.Split(stem, "-")
	cmd := &Command{
		Group:  tokens[0],
		Name:   strings.Join(tokens[1:], " "),
		Binary: path,
	}
	applyMetadata(cmd, readMetadata(path))
	return cmd
}

func applyMetadata(cmd *Command, meta map[string]string) {
	for key, value := range meta {
		switch key {
		case "summary":
			cmd.Summary = value
		case "args":
			cmd.Args = value
		case "examples":
			cmd.Examples = splitPipe(value)
		case "aliases":
			cmd.Aliases = splitPipe(value)
		case "platforms":
			cmd.Platforms = splitPipe(value)
		case "route":
			// An explicit route wins over the name-derived one, for commands
			// whose own words contain a hyphen (`theme bg-switcher`).
			group, name, _ := strings.Cut(strings.TrimSpace(value), " ")
			cmd.Group, cmd.Name = group, name
		case "hidden":
			cmd.Hidden = isTrue(value)
		case "sudo":
			cmd.Sudo = isTrue(value)
		}
	}
	if cmd.Summary == "" {
		cmd.Summary = "(no summary)"
	}
}

// readMetadata scans the head of a file for `# agentic-os:key=value` comments.
func readMetadata(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	meta := map[string]string{}
	scanner := bufio.NewScanner(file)
	for line := 0; line < metadataScanLimit && scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		text = strings.TrimLeft(text, "#/;'\" \t")
		if !strings.HasPrefix(text, metaKey) {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimPrefix(text, metaKey), "=")
		if !ok {
			continue
		}
		meta[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return meta
}

func splitPipe(value string) []string {
	var out []string
	for _, part := range strings.Split(value, "|") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe", ".bat", ".cmd", ".ps1", ".com":
			return true
		}
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

// runExternal executes a discovered plugin, passing our stdio through.
func runExternal(c *Ctx, cmd *Command, args []string) int {
	child := exec.Command(cmd.Binary, args...)
	child.Stdin, child.Stdout, child.Stderr = c.Stdin, c.Stdout, c.Stderr
	child.Env = append(os.Environ(),
		"AGENTIC_OS=1",
		"AGENTIC_OS_VERSION="+c.Version,
		"AGENTIC_OS_ROUTE="+cmd.Route(),
	)
	if err := child.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		c.Warnf("aos: %s: %v\n", cmd.Route(), err)
		return 1
	}
	return 0
}
