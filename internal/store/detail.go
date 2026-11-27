package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DetailFileName is the file inside the data folder that remembers the
// Detail the user picked in each terminal, apart from the Fish, so a Release
// keeps it.
const DetailFileName = "detail.json"

// LoadDetail is the Detail last picked in terminal, as SaveDetail wrote it,
// or false if none was, or the file can't be read: then the Tank opens in
// the Detail guessed for the terminal.
func LoadDetail(dir, terminal string) (string, bool) {
	details, err := loadDetails(dir)
	if err != nil {
		return "", false
	}
	d, ok := details[terminal]
	return d, ok
}

// SaveDetail remembers d as the Detail picked in terminal, keeping what was
// picked in every other terminal. A file it can't read is replaced.
func SaveDetail(dir, terminal, d string) error {
	details, err := loadDetails(dir)
	if err != nil {
		details = map[string]string{}
	}
	details[terminal] = d
	raw, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(dir, DetailFileName, raw)
}

// loadDetails reads the Detail picked in each terminal, by its name.
func loadDetails(dir string) (map[string]string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, DetailFileName))
	if err != nil {
		return nil, err
	}
	details := map[string]string{}
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, err
	}
	return details, nil
}
