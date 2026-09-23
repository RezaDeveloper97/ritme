package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

// Allowlist is contract/allowlist/<group>.yaml: documented, accepted differences.
// One file per group so parallel domain tasks never edit the same file.
type Allowlist struct {
	Group   string       `yaml:"group"`
	Entries []AllowEntry `yaml:"entries"`
}

// AllowEntry suppresses matching mismatches.
type AllowEntry struct {
	// Case is a glob on the case id ("enums.*.ar"); "*" for all cases of the group.
	Case string `yaml:"case"`
	// Step limits the entry to one step (1-based); 0 = any.
	Step int `yaml:"step"`
	// Where is "status", "header:<name>", "body" (any body difference) or a JSON path
	// selector with the ignore syntax ("data.items[*].label", "**.created_at").
	Where string `yaml:"where"`
	// Deviation is the docs/go-migration/deviations.md id (D-nn) that approves this.
	Deviation string `yaml:"deviation"`
	Reason    string `yaml:"reason"`

	rule *ojson.Rule
}

var deviationID = regexp.MustCompile(`^D-\d{2,}$`)

// LoadAllowlist reads the group's allow-list (a missing file is an empty list) and
// checks every entry references a deviation that exists in deviations.md.
func LoadAllowlist(root, group, deviationsPath string) (*Allowlist, error) {
	path := filepath.Join(root, "allowlist", group+".yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // G304: repo-local allow-list
	if errors.Is(err, os.ErrNotExist) {
		return &Allowlist{Group: group}, nil
	}
	if err != nil {
		return nil, err
	}
	var al Allowlist
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&al); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if al.Group != "" && al.Group != group {
		return nil, fmt.Errorf("%s: group %q must be %q", path, al.Group, group)
	}
	devs, err := os.ReadFile(deviationsPath) //nolint:gosec // G304: repo doc
	if err != nil {
		return nil, fmt.Errorf("allow-list needs %s: %w", deviationsPath, err)
	}
	for i := range al.Entries {
		e := &al.Entries[i]
		if !deviationID.MatchString(e.Deviation) {
			return nil, fmt.Errorf("%s: entry %d: deviation %q is not a D-nn id", path, i+1, e.Deviation)
		}
		if !strings.Contains(string(devs), "| "+e.Deviation+" |") {
			return nil, fmt.Errorf("%s: entry %d: %s is not in %s", path, i+1, e.Deviation, deviationsPath)
		}
		if e.Case == "" || e.Where == "" {
			return nil, fmt.Errorf("%s: entry %d: case and where are required", path, i+1)
		}
		if _, err := filepath.Match(e.Case, ""); err != nil {
			return nil, fmt.Errorf("%s: entry %d: bad case glob: %w", path, i+1, err)
		}
		if e.Where != "status" && e.Where != "body" && !strings.HasPrefix(e.Where, "header:") {
			r, err := ojson.ParseRule(e.Where)
			if err != nil {
				return nil, fmt.Errorf("%s: entry %d: %w", path, i+1, err)
			}
			e.rule = &r
		}
	}
	return &al, nil
}

// Allowed returns the entry that covers m in case id, or nil.
func (al *Allowlist) Allowed(caseID string, m Mismatch) *AllowEntry {
	for i := range al.Entries {
		e := &al.Entries[i]
		if ok, _ := filepath.Match(e.Case, caseID); !ok {
			continue
		}
		if e.Step != 0 && e.Step != m.Step {
			continue
		}
		switch {
		case e.Where == "status":
			if m.Kind == "status" {
				return e
			}
		case e.Where == "body":
			if m.Kind == "body" {
				return e
			}
		case strings.HasPrefix(e.Where, "header:"):
			if m.Kind == "header" && strings.EqualFold(e.Where, m.Where) {
				return e
			}
		case e.rule != nil:
			if m.Kind == "body" && m.Path != nil && e.rule.Match(m.Path) {
				return e
			}
		}
	}
	return nil
}
