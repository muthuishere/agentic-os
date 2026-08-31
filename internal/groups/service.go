package groups

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/muthuishere/agentic-os/internal/cli"
)

// servicePrefix namespaces every service aos installs.
//
// This is the safety property of the whole group: `service list` enumerates the
// platform's service store and keeps only labels carrying this prefix, and every
// verb addresses a service by prefixing the name the user typed. A typo can
// therefore never bootout a login agent, disable a systemd unit, or delete a
// scheduled task that belongs to something else.
const servicePrefix = "agentic-os."

// serviceState is what `status` reports and what `list` shows per row.
type serviceState string

const (
	serviceRunning      serviceState = "running"
	serviceStopped      serviceState = "stopped"
	serviceNotInstalled serviceState = "not-installed"
)

// serviceSpec is a service as aos describes it, before any platform
// turns it into a plist, a unit file, or a scheduled task.
type serviceSpec struct {
	Name      string   // short name the user typed, prefix stripped
	Label     string   // namespaced identifier the OS stores
	Command   []string // argv, program resolved to an absolute path
	Dir       string   // working directory
	Autostart bool     // start at login
	OutLog    string
	ErrLog    string
}

// serviceInfo is one row of `list` and the body of `status`.
type serviceInfo struct {
	State  serviceState
	Detail string // pid, exit status, whatever the platform knows
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("service", "Run a command as a managed per-user OS service")
		r.Add(
			&cli.Command{
				Group: "service", Name: "create",
				Summary: "Install a command as a service",
				Args:    "<name> [--autostart] [--now] -- <command> [args...]",
				Examples: []string{
					"aos service create mcp --autostart --now -- aos serve mcp",
					"aos service create nap -- /bin/sleep 60",
				},
				Run: runServiceCreate,
			},
			&cli.Command{
				Group: "service", Name: "remove",
				Summary:  "Stop a service and delete its definition",
				Args:     "<name>",
				Examples: []string{"aos service remove mcp"},
				Run:      runServiceRemove,
			},
			&cli.Command{
				Group: "service", Name: "start",
				Summary:  "Start an installed service",
				Args:     "<name>",
				Examples: []string{"aos service start mcp"},
				Run:      runServiceStart,
			},
			&cli.Command{
				Group: "service", Name: "stop",
				Summary:  "Stop a running service",
				Args:     "<name>",
				Examples: []string{"aos service stop mcp"},
				Run:      runServiceStop,
			},
			&cli.Command{
				Group: "service", Name: "status",
				Summary:  "Report whether a service is running; exits non-zero when it is not",
				Args:     "<name>",
				Examples: []string{"aos service status mcp"},
				Run:      runServiceStatus,
			},
			&cli.Command{
				Group: "service", Name: "list",
				Summary:  "List the services aos manages",
				Examples: []string{"aos service list"},
				Run:      runServiceList,
			},
		)
	})
}

// serviceName validates a user-supplied name and returns it without the
// namespace prefix, so `mcp` and `agentic-os.mcp` both address one service.
func serviceName(raw string) (string, error) {
	name := strings.TrimPrefix(strings.TrimSpace(raw), servicePrefix)
	if name == "" {
		return "", fmt.Errorf("service name is empty")
	}
	if len(name) > 64 {
		return "", fmt.Errorf("service name %q is too long (64 characters max)", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return "", fmt.Errorf("service name %q may only contain letters, digits, dot, dash, and underscore", name)
		}
	}
	// The label is pasted into a file path on every platform, so a name that
	// could walk out of its directory is rejected before it gets there.
	if strings.HasPrefix(name, ".") || strings.Contains(name, "..") {
		return "", fmt.Errorf("service name %q may not start with a dot or contain %q", name, "..")
	}
	return name, nil
}

// serviceLabel is the namespaced identifier the OS stores.
func serviceLabel(name string) string { return servicePrefix + name }

// serviceShortName reverses serviceLabel, reporting false for anything outside
// our namespace so a listing can drop foreign services without inspecting them.
func serviceShortName(label string) (string, bool) {
	if !strings.HasPrefix(label, servicePrefix) {
		return "", false
	}
	name := strings.TrimPrefix(label, servicePrefix)
	if name == "" {
		return "", false
	}
	return name, true
}

// splitServiceCommand cuts argv at the first `--`, which is how a caller stops
// aos from claiming flags that belong to the service's own command.
// With no separator everything is ours to parse, so a stray `--flag` surfaces
// as an unknown flag instead of being silently swallowed into the service.
func splitServiceCommand(args []string) (head, command []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

// serviceLogDir holds the stdout and stderr of every managed service. Runtime
// data lives under the user's config dir, never in a repo.
func serviceLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "agentic-os", "services"), nil
}

// newServiceSpec resolves a command line into something a service manager can
// run. The program is looked up on PATH here because launchd, systemd, and
// schtasks all start their children with a minimal environment and no shell: a
// bare `agentic-os` that works in a terminal would simply never start.
func newServiceSpec(name string, command []string, autostart bool) (serviceSpec, error) {
	if len(command) == 0 {
		return serviceSpec{}, fmt.Errorf("`service create` needs a command after `--`")
	}
	program, err := exec.LookPath(command[0])
	if err != nil {
		return serviceSpec{}, fmt.Errorf("cannot find %q on PATH; give an absolute path", command[0])
	}
	if abs, err := filepath.Abs(program); err == nil {
		program = abs
	}
	dir, err := os.Getwd()
	if err != nil {
		dir = ""
	}
	logDir, err := serviceLogDir()
	if err != nil {
		return serviceSpec{}, err
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return serviceSpec{}, err
	}
	return serviceSpec{
		Name:      name,
		Label:     serviceLabel(name),
		Command:   append([]string{program}, command[1:]...),
		Dir:       dir,
		Autostart: autostart,
		OutLog:    filepath.Join(logDir, name+".out.log"),
		ErrLog:    filepath.Join(logDir, name+".err.log"),
	}, nil
}

func runServiceCreate(c *cli.Ctx, args []string) error {
	head, command := splitServiceCommand(args)
	set, err := parseArgs(head)
	if err != nil {
		return err
	}
	if err := set.Reject("autostart", "now"); err != nil {
		return err
	}
	if len(set.Rest) == 0 {
		return fmt.Errorf("`service create` needs a name")
	}
	name, err := serviceName(set.Rest[0])
	if err != nil {
		return err
	}
	// Without a separator the remaining positionals are the command, which
	// keeps the flagless `service create nap /bin/sleep 60` form working.
	if len(command) == 0 {
		command = set.Rest[1:]
	} else if len(set.Rest) > 1 {
		return fmt.Errorf("unexpected argument %q before `--`", set.Rest[1])
	}

	spec, err := newServiceSpec(name, command, set.Has("autostart"))
	if err != nil {
		return err
	}
	if info, err := serviceQuery(spec.Label); err == nil && info.State != serviceNotInstalled {
		return &cli.ExitError{Code: 1,
			Message: fmt.Sprintf("service %q already exists; remove it first", name)}
	}

	notes, err := serviceInstall(spec)
	if err != nil {
		return err
	}
	c.Printf("name       %s\n", spec.Name)
	c.Printf("label      %s\n", spec.Label)
	c.Printf("command    %s\n", strings.Join(spec.Command, " "))
	c.Printf("logs       %s\n", spec.OutLog)
	c.Printf("autostart  %s\n", yesNo(spec.Autostart))

	if set.Has("now") {
		if err := serviceStartOne(spec.Label); err != nil {
			return err
		}
		c.Println("state      running")
	} else {
		c.Println("state      installed (not started)")
	}
	for _, note := range notes {
		c.Printf("note       %s\n", note)
	}
	return nil
}

func runServiceRemove(c *cli.Ctx, args []string) error {
	label, err := serviceTarget(args, "remove")
	if err != nil {
		return err
	}
	if err := serviceUninstall(label); err != nil {
		return err
	}
	// The logs are ours and namespaced, so removing the service removes them
	// too; otherwise every create/remove cycle leaves litter behind.
	if name, ok := serviceShortName(label); ok {
		removeServiceLogs(name)
	}
	c.Printf("removed %s\n", label)
	return nil
}

// removeServiceLogs deletes a gone service's log files, best-effort: failing to
// tidy up is not a reason to report a removal that did happen as failed.
func removeServiceLogs(name string) {
	dir, err := serviceLogDir()
	if err != nil {
		return
	}
	for _, suffix := range []string{".out.log", ".err.log"} {
		_ = os.Remove(filepath.Join(dir, name+suffix))
	}
}

func runServiceStart(c *cli.Ctx, args []string) error {
	label, err := serviceTarget(args, "start")
	if err != nil {
		return err
	}
	if err := serviceStartOne(label); err != nil {
		return err
	}
	c.Printf("started %s\n", label)
	return nil
}

func runServiceStop(c *cli.Ctx, args []string) error {
	label, err := serviceTarget(args, "stop")
	if err != nil {
		return err
	}
	if err := serviceStopOne(label); err != nil {
		return err
	}
	c.Printf("stopped %s\n", label)
	return nil
}

func runServiceStatus(c *cli.Ctx, args []string) error {
	label, err := serviceTarget(args, "status")
	if err != nil {
		return err
	}
	info, err := serviceQuery(label)
	if err != nil {
		return err
	}
	name, _ := serviceShortName(label)
	c.Printf("name    %s\n", name)
	c.Printf("label   %s\n", label)
	c.Printf("state   %s\n", info.State)
	if info.Detail != "" {
		c.Printf("detail  %s\n", info.Detail)
	}
	// Exiting non-zero for anything but "running" is what makes this compose in
	// a health check: `aos service status mcp || aos service start mcp`.
	if info.State != serviceRunning {
		return &cli.ExitError{Code: 1}
	}
	return nil
}

func runServiceList(c *cli.Ctx, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("`service list` takes no arguments")
	}
	labels, err := serviceLabels()
	if err != nil {
		return err
	}
	sort.Strings(labels)

	type row struct{ name, state, detail string }
	var rows []row
	width := 0
	for _, label := range labels {
		name, ok := serviceShortName(label)
		if !ok {
			continue
		}
		entry := row{name: name, state: "unknown"}
		if info, err := serviceQuery(label); err == nil {
			entry.state, entry.detail = string(info.State), info.Detail
		}
		rows = append(rows, entry)
		if len(name) > width {
			width = len(name)
		}
	}
	if len(rows) == 0 {
		c.Println("no services")
		return nil
	}
	for _, entry := range rows {
		if entry.detail != "" {
			c.Printf("%-*s  %-13s %s\n", width, entry.name, entry.state, entry.detail)
			continue
		}
		c.Printf("%-*s  %s\n", width, entry.name, entry.state)
	}
	return nil
}

// serviceTarget parses the single-name argument every verb but `create` takes,
// returning the namespaced label.
func serviceTarget(args []string, verb string) (string, error) {
	set, err := parseArgs(args)
	if err != nil {
		return "", err
	}
	if err := set.Reject(); err != nil {
		return "", err
	}
	if len(set.Rest) != 1 {
		return "", fmt.Errorf("`service %s` needs exactly one service name", verb)
	}
	name, err := serviceName(set.Rest[0])
	if err != nil {
		return "", err
	}
	return serviceLabel(name), nil
}

// renderLaunchdPlist builds a launchd user agent. KeepAlive stays off: a
// service aos stopped should stay stopped, and a respawning job would
// make `stop` a lie.
func renderLaunchdPlist(spec serviceSpec) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>" + xmlText(spec.Label) + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range spec.Command {
		b.WriteString("\t\t<string>" + xmlText(arg) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	if spec.Dir != "" {
		b.WriteString("\t<key>WorkingDirectory</key>\n\t<string>" + xmlText(spec.Dir) + "</string>\n")
	}
	b.WriteString("\t<key>StandardOutPath</key>\n\t<string>" + xmlText(spec.OutLog) + "</string>\n")
	b.WriteString("\t<key>StandardErrorPath</key>\n\t<string>" + xmlText(spec.ErrLog) + "</string>\n")
	b.WriteString("\t<key>RunAtLoad</key>\n\t<" + boolTag(spec.Autostart) + "/>\n")
	b.WriteString("\t<key>KeepAlive</key>\n\t<false/>\n")
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

// renderSystemdUnit builds a systemd --user service. Type=simple with no
// Restart mirrors launchd's KeepAlive=false, so `stop` means the same thing on
// both platforms.
func renderSystemdUnit(spec serviceSpec) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=aos service " + spec.Name + "\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("ExecStart=" + systemdCommand(spec.Command) + "\n")
	if spec.Dir != "" {
		b.WriteString("WorkingDirectory=" + spec.Dir + "\n")
	}
	b.WriteString("StandardOutput=append:" + spec.OutLog + "\n")
	b.WriteString("StandardError=append:" + spec.ErrLog + "\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=default.target\n")
	return b.String()
}

// systemdCommand renders argv for ExecStart. systemd splits the line on
// whitespace and honours double quotes, so every argument is quoted rather
// than the renderer guessing which ones need it.
func systemdCommand(argv []string) string {
	escape := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	quoted := make([]string, 0, len(argv))
	for _, arg := range argv {
		quoted = append(quoted, `"`+escape.Replace(arg)+`"`)
	}
	return strings.Join(quoted, " ")
}

// schtasksCommand renders argv into the single /TR string schtasks accepts.
func schtasksCommand(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		if arg == "" || strings.ContainsAny(arg, " \t\"") {
			parts = append(parts, `"`+strings.ReplaceAll(arg, `"`, `\"`)+`"`)
			continue
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func xmlText(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

func boolTag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
