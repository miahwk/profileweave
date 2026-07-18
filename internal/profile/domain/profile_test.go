package domain

import (
	"errors"
	"testing"
	"time"

	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
)

func validInput() Input {
	return Input{
		Name: "QA profile", Tags: []string{"QA", "qa", "China"}, StartURL: "https://example.test/path",
		Browser: Browser{Kind: "auto"}, Fingerprint: fingerprint.Default(), Proxy: fingerprint.Proxy{Mode: "direct"},
	}
}

func TestNewAndUpdateProfile(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 2, 3, 0, time.FixedZone("test", 8*60*60))
	profile, err := New("p_0123456789abcdef0123456789abcdef", validInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Revision != 1 || !profile.CreatedAt.Equal(now.UTC()) || len(profile.Tags) != 2 {
		t.Fatalf("unexpected new profile: %#v", profile)
	}
	in := profile.Input()
	in.Name = " Updated "
	updated, err := profile.Update(in, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated" || updated.Revision != 2 || !updated.CreatedAt.Equal(profile.CreatedAt) {
		t.Fatalf("unexpected updated profile: %#v", updated)
	}
}

func TestProfileRejectsUnsafeBoundaries(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Input)
	}{
		{"empty name", func(in *Input) { in.Name = "" }},
		{"unsafe URL", func(in *Input) { in.StartURL = "file:///etc/passwd" }},
		{"URL credentials", func(in *Input) { in.StartURL = "https://user:pass@example.com" }},
		{"unknown browser", func(in *Input) { in.Browser.Kind = "shell" }},
		{"custom browser disabled", func(in *Input) { in.Browser.Kind = "custom" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validInput()
			tc.edit(&input)
			_, err := New("p_0123456789abcdef0123456789abcdef", input, time.Now())
			var validation *ValidationError
			if !errors.As(err, &validation) || len(validation.Details) == 0 {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestProfileIDIsTraversalSafe(t *testing.T) {
	for _, id := range []string{"../profile", "p_123", "p_0123456789abcdef0123456789abcdeg", ""} {
		if ValidID(id) {
			t.Fatalf("accepted unsafe ID %q", id)
		}
	}
}
