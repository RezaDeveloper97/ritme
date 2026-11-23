package testdb

import "testing"

func TestMissingIsFatal(t *testing.T) {
	cases := []struct {
		name      string
		ci, opt   string
		wantFatal bool
	}{
		{"local run, no CI", "", "", false},
		{"GitHub Actions", "true", "", true},
		{"CI=1", "1", "", true},
		{"CI with spaces", " true ", "", true},
		{"CI explicitly false", "false", "", false},
		{"CI garbage is not CI", "yes-please", "", false},
		{"unit-only CI job opts out", "true", "1", false},
		{"opt-out set to false keeps it fatal", "true", "false", true},
		{"opt-out without CI", "", "1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"CI": tc.ci, "TESTDB_OPTIONAL": tc.opt}
			if got := missingIsFatal(func(k string) string { return env[k] }); got != tc.wantFatal {
				t.Fatalf("missingIsFatal(CI=%q, TESTDB_OPTIONAL=%q) = %v, want %v", tc.ci, tc.opt, got, tc.wantFatal)
			}
		})
	}
}
