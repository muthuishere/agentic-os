package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// ConfigDir is where a user's own additions live: adapters, plugin binaries,
// and anything else aos reads but does not ship.
func ConfigDir(env func(string) string) string {
	if dir := strings.TrimSpace(env("AGENTIC_OS_CONFIG_DIR")); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agentic-os")
	}
	return filepath.Join(home, ".config", "agentic-os")
}

// AdapterDir holds the JSON files that define user commands.
func AdapterDir(env func(string) string) string {
	return filepath.Join(ConfigDir(env), "adapters")
}

// Adapter is one JSON file: a group and the commands it adds.
//
// This is the low-ceremony half of extension. A plugin is an executable and can
// do anything; an adapter is a few lines of JSON wrapping a command line
// someone already runs, which is what most additions actually are.
type Adapter struct {
	Group       string           `json:"group"`
	Description string           `json:"description"`
	Commands    []AdapterCommand `json:"commands"`

	// Path is where this adapter was read from, for `adapters list`.
	Path string `json:"-"`
}

// AdapterCommand is one command an adapter contributes.
type AdapterCommand struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Args     string   `json:"args,omitempty"`
	Examples []string `json:"examples,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
	// Run is a command line executed by the platform shell. The user's own
	// arguments are appended to it, quoted.
	Run string `json:"run"`
	// Env adds variables for this command only.
	Env map[string]string `json:"env,omitempty"`
	// Platforms restricts the command; empty means everywhere.
	Platforms []string `json:"platforms,omitempty"`
	// NeedsDisplay marks a command that drives the GUI.
	NeedsDisplay bool `json:"needsDisplay,omitempty"`
	// Blocking marks a command that runs until interrupted, keeping it out of
	// the MCP tool list.
	Blocking bool `json:"blocking,omitempty"`
	Hidden   bool `json:"hidden,omitempty"`
}

// LoadAdapters registers every adapter-defined command it can find.
//
// Builtins win, exactly as they do over plugins: an adapter cannot shadow a
// shipped command and silently change what it does.
func LoadAdapters(r *Registry, env func(string) string) []Adapter {
	dir := AdapterDir(env)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			names = append(names, entry.Name())
		}
	}
	// Deterministic order, so two adapters claiming the same route always
	// resolve the same way rather than depending on directory order.
	sort.Strings(names)

	var loaded []Adapter
	for _, name := range names {
		path := filepath.Join(dir, name)
		adapter, err := readAdapter(path)
		if err != nil {
			r.warnings = append(r.warnings, fmt.Sprintf("adapter %s: %v", name, err))
			continue
		}
		registerAdapter(r, adapter)
		loaded = append(loaded, *adapter)
	}
	return loaded
}

func readAdapter(path string) (*Adapter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var adapter Adapter
	if err := json.Unmarshal(data, &adapter); err != nil {
		return nil, err
	}
	if adapter.Group == "" {
		return nil, fmt.Errorf("needs a group")
	}
	if strings.ContainsAny(adapter.Group, " \t/\\") {
		return nil, fmt.Errorf("group %q cannot contain spaces or path separators", adapter.Group)
	}
	for i, cmd := range adapter.Commands {
		if cmd.Run == "" {
			return nil, fmt.Errorf("command %d (%q) needs a run", i, cmd.Name)
		}
		if cmd.Summary == "" {
			return nil, fmt.Errorf("command %d (%q) needs a summary", i, cmd.Name)
		}
	}
	adapter.Path = path
	return &adapter, nil
}

func registerAdapter(r *Registry, adapter *Adapter) {
	if r.Group(adapter.Group) == nil || r.Group(adapter.Group).Description == "" {
		description := adapter.Description
		if description == "" {
			description = "User commands from " + filepath.Base(adapter.Path)
		}
		r.Describe(adapter.Group, description)
	}

	for _, definition := range adapter.Commands {
		if r.Lookup(adapter.Group, definition.Name) != nil {
			continue
		}
		spec := definition // captured per iteration
		r.Add(&Command{
			Group:        adapter.Group,
			Name:         spec.Name,
			Summary:      spec.Summary,
			Args:         spec.Args,
			Examples:     spec.Examples,
			Aliases:      spec.Aliases,
			Platforms:    spec.Platforms,
			NeedsDisplay: spec.NeedsDisplay,
			Blocking:     spec.Blocking,
			Hidden:       spec.Hidden,
			Run: func(c *Ctx, args []string) error {
				return runAdapterCommand(c, spec, args)
			},
		})
	}
}

// runAdapterCommand executes the adapter's command line through the platform
// shell, with the user's arguments appended and quoted.
func runAdapterCommand(c *Ctx, spec AdapterCommand, args []string) error {
	line := spec.Run
	if len(args) > 0 {
		line += " " + quoteArgs(args)
	}

	shell := sys.Shell()
	env := os.Environ()
	for key, value := range spec.Env {
		env = append(env, key+"="+value)
	}

	if err := sys.PassthroughEnv(env, shell[0], append(shell[1:], line)...); err != nil {
		if code := sys.ExitCodeOf(err); code > 0 {
			return &ExitError{Code: code}
		}
		return err
	}
	return nil
}

// quoteArgs makes arguments safe for the shell that will re-parse them.
func quoteArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "" && !strings.ContainsAny(arg, " \t\"'$`\\|&;<>()*?[]{}!#~") {
			quoted = append(quoted, arg)
			continue
		}
		quoted = append(quoted, `"`+strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`).Replace(arg)+`"`)
	}
	return strings.Join(quoted, " ")
}
