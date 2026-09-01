package groups

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/muthuishere/aos/internal/cli"
)

// fileInfoJSON is the shape `file stat` and `file list --json` emit.
type fileInfoJSON struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Dir      bool   `json:"dir"`
	Modified string `json:"modified"`
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("file", "Read, write, and inspect files")
		r.Add(
			&cli.Command{
				Group: "file", Name: "read",
				Summary: "Print a file, optionally a line range",
				Args:    "<path> [--lines=<from>:<to>]",
				Examples: []string{
					"aos file read go.mod",
					"aos file read main.go --lines=10:40",
				},
				Run: runFileRead,
			},
			&cli.Command{
				Group: "file", Name: "write",
				Summary: "Write stdin, or the given words, to a file",
				Args:    "<path> [text...]",
				Examples: []string{
					`aos file write notes.txt "first line"`,
					"cat in.txt | aos file write out.txt",
				},
				Run: func(c *cli.Ctx, args []string) error { return writeFile(c, args, false) },
			},
			&cli.Command{
				Group: "file", Name: "append",
				Summary:  "Append stdin, or the given words, to a file",
				Args:     "<path> [text...]",
				Examples: []string{`aos file append log.txt "done"`},
				Run:      func(c *cli.Ctx, args []string) error { return writeFile(c, args, true) },
			},
			&cli.Command{
				Group: "file", Name: "list",
				Summary: "List a directory",
				Args:    "[path] [--all] [--json]",
				Examples: []string{
					"aos file list",
					"aos file list ~/Downloads --json",
				},
				Run: runFileList,
			},
			&cli.Command{
				Group: "file", Name: "stat",
				Summary:  "Print one path's size, mode, and modified time",
				Args:     "<path> [--json]",
				Examples: []string{"aos file stat go.mod --json"},
				Run:      runFileStat,
			},
			&cli.Command{
				Group: "file", Name: "mkdir",
				Summary:  "Create a directory, including parents",
				Args:     "<path>",
				Examples: []string{"aos file mkdir ~/.config/aos/layouts"},
				Run:      runFileMkdir,
			},
			&cli.Command{
				Group: "file", Name: "copy",
				Summary:  "Copy a file",
				Args:     "<source> <destination>",
				Examples: []string{"aos file copy a.txt b.txt"},
				Run:      runFileCopy,
			},
			&cli.Command{
				Group: "file", Name: "move",
				Summary:  "Move or rename a path",
				Args:     "<source> <destination>",
				Examples: []string{"aos file move old.txt new.txt"},
				Run:      runFileMove,
			},
			&cli.Command{
				Group: "file", Name: "delete",
				Summary:  "Delete a file, or a directory with --recursive",
				Args:     "<path> [--recursive]",
				Examples: []string{"aos file delete /tmp/scratch --recursive"},
				Run:      runFileDelete,
			},
		)
	})
}

func runFileRead(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "lines")
	if err != nil {
		return err
	}
	if err := set.Reject("lines"); err != nil {
		return err
	}
	if len(set.Rest) != 1 {
		return fmt.Errorf("`file read` takes one path")
	}

	data, err := os.ReadFile(expandHome(set.Rest[0]))
	if err != nil {
		return err
	}
	spec := set.String("lines", "")
	if spec == "" {
		_, err := c.Stdout.Write(data)
		return err
	}

	from, to, err := parseLineRange(spec)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if from > len(lines) {
		return nil
	}
	if to > len(lines) {
		to = len(lines)
	}
	c.Println(strings.Join(lines[from-1:to], "\n"))
	return nil
}

// parseLineRange reads the 1-indexed, inclusive `from:to` form.
func parseLineRange(spec string) (int, int, error) {
	fromText, toText, ok := strings.Cut(spec, ":")
	if !ok {
		return 0, 0, fmt.Errorf("--lines wants <from>:<to>, got %q", spec)
	}
	from, err := parseInt(fromText)
	if err != nil || from < 1 {
		return 0, 0, fmt.Errorf("--lines start must be 1 or more, got %q", fromText)
	}
	to, err := parseInt(toText)
	if err != nil || to < from {
		return 0, 0, fmt.Errorf("--lines end must be at least the start, got %q", toText)
	}
	return from, to, nil
}

func writeFile(c *cli.Ctx, args []string, appending bool) error {
	if len(args) == 0 {
		return fmt.Errorf("needs a path")
	}
	path := expandHome(args[0])

	var content string
	if len(args) > 1 {
		content = strings.Join(args[1:], " ") + "\n"
	} else {
		piped, err := io.ReadAll(c.Stdin)
		if err != nil {
			return err
		}
		content = string(piped)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if appending {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return err
	}
	c.Printf("%s  %d bytes\n", path, len(content))
	return nil
}

func runFileList(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("all", "json"); err != nil {
		return err
	}

	dir := "."
	if len(set.Rest) == 1 {
		dir = expandHome(set.Rest[0])
	} else if len(set.Rest) > 1 {
		return fmt.Errorf("`file list` takes at most one path")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var infos []fileInfoJSON
	for _, entry := range entries {
		if !set.Has("all") && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		infos = append(infos, toFileInfo(filepath.Join(dir, entry.Name()), info))
	}

	if set.Has("json") {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}
	for _, info := range infos {
		kind := "     "
		if info.Dir {
			kind = "dir  "
		}
		c.Printf("%s%10d  %s  %s\n", kind, info.Size, info.Modified, info.Name)
	}
	return nil
}

func runFileStat(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("json"); err != nil {
		return err
	}
	if len(set.Rest) != 1 {
		return fmt.Errorf("`file stat` takes one path")
	}

	path := expandHome(set.Rest[0])
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	payload := toFileInfo(path, info)

	if set.Has("json") {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	c.Printf("path      %s\n", payload.Path)
	c.Printf("size      %d\n", payload.Size)
	c.Printf("mode      %s\n", payload.Mode)
	c.Printf("dir       %t\n", payload.Dir)
	c.Printf("modified  %s\n", payload.Modified)
	return nil
}

func toFileInfo(path string, info fs.FileInfo) fileInfoJSON {
	return fileInfoJSON{
		Name:     info.Name(),
		Path:     path,
		Size:     info.Size(),
		Mode:     info.Mode().String(),
		Dir:      info.IsDir(),
		Modified: info.ModTime().Format(time.RFC3339),
	}
}

func runFileMkdir(c *cli.Ctx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("`file mkdir` takes one path")
	}
	path := expandHome(args[0])
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	c.Println(path)
	return nil
}

func runFileCopy(c *cli.Ctx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("`file copy` takes a source and a destination")
	}
	source, destination := expandHome(args[0]), expandHome(args[1])

	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory; `file copy` handles single files", source)
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, in)
	if err != nil {
		return err
	}
	c.Printf("%s  %d bytes\n", destination, written)
	return nil
}

func runFileMove(c *cli.Ctx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("`file move` takes a source and a destination")
	}
	source, destination := expandHome(args[0]), expandHome(args[1])
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		return err
	}
	c.Println(destination)
	return nil
}

func runFileDelete(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("recursive"); err != nil {
		return err
	}
	if len(set.Rest) != 1 {
		return fmt.Errorf("`file delete` takes one path")
	}

	path, err := filepath.Abs(expandHome(set.Rest[0]))
	if err != nil {
		return err
	}
	// Deleting a filesystem root or a home directory is never what was meant,
	// and the mistake is unrecoverable, so refuse outright.
	if isProtectedPath(path) {
		return fmt.Errorf("refusing to delete %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if !set.Has("recursive") {
			return fmt.Errorf("%s is a directory; pass --recursive to delete it and its contents", path)
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else if err := os.Remove(path); err != nil {
		return err
	}
	c.Printf("deleted %s\n", path)
	return nil
}

// isProtectedPath guards the paths whose deletion would be catastrophic: a
// filesystem or volume root, and the user's home directory itself.
func isProtectedPath(path string) bool {
	cleaned := filepath.Clean(path)
	if cleaned == string(filepath.Separator) || cleaned == filepath.VolumeName(cleaned)+string(filepath.Separator) {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == cleaned {
		return true
	}
	// A direct child of a filesystem root is a system directory — /etc, /usr,
	// /Users, C:\Windows. Deleting one is unrecoverable and never what a caller
	// meant, and `file delete` is reachable by an agent over MCP, so the guard
	// has to hold without a human reading the path first.
	//
	// The previous test here was `filepath.Dir(cleaned) == cleaned`, which is
	// only ever true for the root itself and so was already covered two lines
	// above: every system directory fell through unguarded.
	parent := filepath.Dir(cleaned)
	return filepath.Dir(parent) == parent
}
