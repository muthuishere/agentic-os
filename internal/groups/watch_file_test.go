package groups

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	t0 = time.Unix(1700000000, 0)
	t1 = time.Unix(1700000060, 0)
)

func file(mod time.Time, size int64) fileEntry { return fileEntry{ModTime: mod, Size: size} }
func dir(mod time.Time) fileEntry              { return fileEntry{ModTime: mod, IsDir: true} }

// The diff is the whole watcher: everything the command prints comes out of
// comparing two snapshots, so the interesting cases are all here rather than
// behind a sleep and a real filesystem.
func TestDiffFileSnapshots(t *testing.T) {
	cases := []struct {
		name string
		prev map[string]fileEntry
		cur  map[string]fileEntry
		want []fileChange
	}{
		{
			name: "nothing changed is no news",
			prev: map[string]fileEntry{"/w/a.txt": file(t0, 3)},
			cur:  map[string]fileEntry{"/w/a.txt": file(t0, 3)},
		},
		{
			name: "a new entry is created",
			prev: map[string]fileEntry{},
			cur:  map[string]fileEntry{"/w/a.txt": file(t0, 3)},
			want: []fileChange{{Kind: "created", Path: "/w/a.txt", Size: 3}},
		},
		{
			name: "a gone entry is deleted",
			prev: map[string]fileEntry{"/w/a.txt": file(t0, 3)},
			cur:  map[string]fileEntry{},
			want: []fileChange{{Kind: "deleted", Path: "/w/a.txt"}},
		},
		{
			name: "a newer modtime is a modification",
			prev: map[string]fileEntry{"/w/a.txt": file(t0, 3)},
			cur:  map[string]fileEntry{"/w/a.txt": file(t1, 3)},
			want: []fileChange{{Kind: "modified", Path: "/w/a.txt", Size: 3}},
		},
		{
			// An editor that rewrites a file within the same modtime tick still
			// changed it, and the size is the cheap second signal.
			name: "a different size is a modification even at the same modtime",
			prev: map[string]fileEntry{"/w/a.txt": file(t0, 3)},
			cur:  map[string]fileEntry{"/w/a.txt": file(t0, 9)},
			want: []fileChange{{Kind: "modified", Path: "/w/a.txt", Size: 9}},
		},
		{
			// A directory's modtime moves whenever a child appears or goes, so
			// tracking it would double-report every create and delete inside it.
			name: "a directory's own modtime is not a change",
			prev: map[string]fileEntry{"/w/sub": dir(t0)},
			cur:  map[string]fileEntry{"/w/sub": dir(t1)},
		},
		{
			name: "a file replaced by a directory is a modification",
			prev: map[string]fileEntry{"/w/x": file(t0, 3)},
			cur:  map[string]fileEntry{"/w/x": dir(t1)},
			want: []fileChange{{Kind: "modified", Path: "/w/x", Dir: true}},
		},
		{
			name: "a new directory is still a creation",
			prev: map[string]fileEntry{},
			cur:  map[string]fileEntry{"/w/sub": dir(t0)},
			want: []fileChange{{Kind: "created", Path: "/w/sub", Dir: true}},
		},
		{
			// Map iteration is random; the output must not be.
			name: "several changes in one tick come out in path order",
			prev: map[string]fileEntry{"/w/b.txt": file(t0, 1), "/w/c.txt": file(t0, 1)},
			cur:  map[string]fileEntry{"/w/a.txt": file(t0, 2), "/w/b.txt": file(t1, 1)},
			want: []fileChange{
				{Kind: "created", Path: "/w/a.txt", Size: 2},
				{Kind: "modified", Path: "/w/b.txt", Size: 1},
				{Kind: "deleted", Path: "/w/c.txt"},
			},
		},
		{
			name: "two empty snapshots",
			prev: map[string]fileEntry{},
			cur:  map[string]fileEntry{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diffFileSnapshots(tc.prev, tc.cur)
			if len(got) != len(tc.want) {
				t.Fatalf("diffFileSnapshots() = %+v; want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("change %d = %+v; want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// snapshotPath is the only part that touches the disk, so it gets one small
// pass over a temp tree rather than a matrix.
func TestSnapshotPath(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"), "hi")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "sub", "b.txt"), "deeper")

	shallow, err := snapshotPath(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(shallow) != 2 {
		t.Fatalf("shallow snapshot = %v; want the two direct entries", keysOf(shallow))
	}
	if _, ok := shallow[filepath.Join(root, "sub", "b.txt")]; ok {
		t.Fatalf("shallow snapshot descended into sub/")
	}

	deep, err := snapshotPath(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := deep[filepath.Join(root, "sub", "b.txt")]; !ok {
		t.Fatalf("recursive snapshot = %v; want sub/b.txt", keysOf(deep))
	}
	if _, ok := deep[root]; ok {
		t.Fatalf("the root is the subject, not an entry in itself")
	}

	// A single file watches only itself.
	one, err := snapshotPath(filepath.Join(root, "a.txt"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[filepath.Join(root, "a.txt")].Size != 2 {
		t.Fatalf("file snapshot = %+v; want just a.txt at 2 bytes", one)
	}

	// A root that disappears mid-run is a deletion to report, not a failure.
	gone, err := snapshotPath(filepath.Join(root, "nope"), true)
	if err != nil || len(gone) != 0 {
		t.Fatalf("snapshotPath(missing) = %v, %v; want an empty snapshot and no error", gone, err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func keysOf(snap map[string]fileEntry) []string {
	names := make([]string, 0, len(snap))
	for path := range snap {
		names = append(names, path)
	}
	return names
}
