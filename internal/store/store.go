// Package store loads and saves the Fish in the user's data folder.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/Kyu-seok/shellfish/internal/fish"
)

// FileName is the save file's name inside the data folder.
const FileName = "fish.json"

// schemaVersion is the save file format this build writes. Version 2 added
// the Species and dropped the colour.
const schemaVersion = 2

// speciesless is the save file format from before Species existed, which
// this build still reads, as a Fish with no Species yet.
const speciesless = 1

// ErrNoFish means the data folder holds no saved Fish: this is a first launch.
var ErrNoFish = errors.New("no saved Fish")

// saved is the save file's JSON shape.
type saved struct {
	Version       int       `json:"version"`
	Name          string    `json:"name"`
	Species       string    `json:"species,omitempty"` // by its stable name
	Born          time.Time `json:"born"`
	Hunger        float64   `json:"hunger"`
	HungerUpdated time.Time `json:"hungerUpdated"`
	LastSeen      time.Time `json:"lastSeen"`
}

// Load reads the saved Fish from dir. It returns ErrNoFish if there is none.
func Load(dir string) (fish.Fish, error) {
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return fish.Fish{}, ErrNoFish
	}
	if err != nil {
		return fish.Fish{}, fmt.Errorf("reading saved Fish: %w", err)
	}
	var s saved
	if err := json.Unmarshal(raw, &s); err != nil {
		return fish.Fish{}, fmt.Errorf("reading saved Fish: %w", err)
	}
	species := fish.NoSpecies
	switch s.Version {
	case speciesless:
	case schemaVersion:
		var ok bool
		if species, ok = fish.SpeciesNamed(s.Species); !ok {
			return fish.Fish{}, fmt.Errorf("saved Fish is of unknown Species %q", s.Species)
		}
	default:
		return fish.Fish{}, fmt.Errorf("saved Fish has schema version %d, this shellfish reads versions %d to %d", s.Version, speciesless, schemaVersion)
	}
	return fish.Fish{
		Name:          s.Name,
		Species:       species,
		Born:          s.Born,
		Hunger:        s.Hunger,
		HungerUpdated: s.HungerUpdated,
		LastSeen:      s.LastSeen,
	}, nil
}

// Save writes f to dir atomically: to a temporary file first, then renamed
// over the save file, so a crash never leaves a half-written Fish. It
// creates dir if needed.
func Save(dir string, f fish.Fish) error {
	raw, err := json.MarshalIndent(saved{
		Version:       schemaVersion,
		Name:          f.Name,
		Species:       f.Species.String(),
		Born:          f.Born,
		Hunger:        f.Hunger,
		HungerUpdated: f.HungerUpdated,
		LastSeen:      f.LastSeen,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("saving Fish: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("saving Fish: %w", err)
	}
	tmp, err := os.CreateTemp(dir, FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("saving Fish: %w", err)
	}
	defer os.Remove(tmp.Name()) // a no-op once renamed
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("saving Fish: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("saving Fish: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("saving Fish: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, FileName)); err != nil {
		return fmt.Errorf("saving Fish: %w", err)
	}
	return nil
}
