package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration describes a SQL migration file discovered on disk.
type Migration struct {
	Name string
	Path string
}

// Discover returns sorted SQL migration files from the provided directory.
func Discover(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		migrations = append(migrations, Migration{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(migrations, func(i int, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}
