package groups

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/muthuishere/agentic-os/internal/cli"
)

// fontExtensions are the font container formats worth listing.
var fontExtensions = map[string]bool{
	".ttf": true, ".otf": true, ".ttc": true, ".otc": true, ".woff": true, ".woff2": true,
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("font", "Installed fonts")
		r.Add(
			&cli.Command{
				Group: "font", Name: "list",
				Summary:  "List installed font families",
				Args:     "[filter]",
				Examples: []string{"agentic-os font list", "agentic-os font list mono"},
				Run:      runFontList,
			},
			&cli.Command{
				Group: "font", Name: "dirs",
				Summary: "Print the directories fonts are read from",
				Run: func(c *cli.Ctx, _ []string) error {
					for _, dir := range fontDirs() {
						c.Println(dir)
					}
					return nil
				},
			},
		)
	})
}

func runFontList(c *cli.Ctx, args []string) error {
	filter := strings.ToLower(strings.Join(args, " "))

	// Some platforms publish real family names ("Arial Bold") and do not make
	// you guess them from file stems ("arialbd"). Prefer that when it exists.
	if names, ok := fontNames(); ok {
		return printFamilies(c, names, filter)
	}

	seen := map[string]bool{}
	for _, dir := range fontDirs() {
		// A font directory that does not exist is normal, not an error: which
		// of the standard locations are present varies by machine.
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if !fontExtensions[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			family := fontFamily(entry.Name())
			if filter != "" && !strings.Contains(strings.ToLower(family), filter) {
				return nil
			}
			seen[family] = true
			return nil
		})
	}

	families := make([]string, 0, len(seen))
	for family := range seen {
		families = append(families, family)
	}
	return printFamilies(c, families, "")
}

// printFamilies de-duplicates, filters, sorts, and prints.
func printFamilies(c *cli.Ctx, names []string, filter string) error {
	seen := map[string]bool{}
	for _, name := range names {
		if name = strings.TrimSpace(name); name == "" {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		seen[name] = true
	}

	families := make([]string, 0, len(seen))
	for name := range seen {
		families = append(families, name)
	}
	sort.Strings(families)
	for _, family := range families {
		c.Println(family)
	}
	return nil
}

// fontStyles are the trailing words dropped to turn a font file name into a
// family name: "JetBrainsMono-BoldItalic.ttf" becomes "JetBrainsMono".
var fontStyles = []string{
	"thin", "extralight", "ultralight", "light", "regular", "book", "medium",
	"semibold", "demibold", "bold", "extrabold", "ultrabold", "black", "heavy",
	"italic", "oblique", "condensed", "expanded", "nerdfont", "nf",
}

func fontFamily(fileName string) string {
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	parts := strings.FieldsFunc(stem, func(r rune) bool { return r == '-' || r == '_' })
	for len(parts) > 1 {
		last := strings.ToLower(parts[len(parts)-1])
		if !isFontStyle(last) {
			break
		}
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return stem
	}
	return strings.Join(parts, " ")
}

// isFontStyle reports whether a token is made up entirely of style words.
// Font files run them together ("BoldItalic", "ExtraLightItalic"), so a single
// equality check is not enough; this consumes style words greedily, longest
// first, and only succeeds if nothing is left over.
func isFontStyle(word string) bool {
	for word != "" {
		match := ""
		for _, style := range fontStyles {
			if len(style) > len(match) && strings.HasPrefix(word, style) {
				match = style
			}
		}
		if match == "" {
			return false
		}
		word = word[len(match):]
	}
	return true
}
