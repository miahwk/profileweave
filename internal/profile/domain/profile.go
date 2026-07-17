package domain

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
)

var (
	ErrNotFound = errors.New("profile not found")
	ErrConflict = errors.New("profile revision conflict")
	idPattern   = regexp.MustCompile(`^p_[a-f0-9]{32}$`)
)

type Browser struct {
	Kind       string `json:"kind"`
	CustomPath string `json:"customPath,omitempty"`
}

type Profile struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Notes       string                  `json:"notes,omitempty"`
	Tags        []string                `json:"tags"`
	StartURL    string                  `json:"startURL"`
	Browser     Browser                 `json:"browser"`
	Fingerprint fingerprint.Fingerprint `json:"fingerprint"`
	Proxy       fingerprint.Proxy       `json:"proxy"`
	Revision    uint64                  `json:"revision"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

type Input struct {
	Name        string                  `json:"name"`
	Notes       string                  `json:"notes,omitempty"`
	Tags        []string                `json:"tags"`
	StartURL    string                  `json:"startURL"`
	Browser     Browser                 `json:"browser"`
	Fingerprint fingerprint.Fingerprint `json:"fingerprint"`
	Proxy       fingerprint.Proxy       `json:"proxy"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct{ Details []FieldError }

func (e *ValidationError) Error() string { return "profile is invalid" }

type Repository interface {
	List(context.Context) ([]Profile, error)
	Get(context.Context, string) (Profile, error)
	Save(context.Context, Profile, uint64) error
	Delete(context.Context, string) error
}

func ValidID(id string) bool { return idPattern.MatchString(id) }

func ValidateInput(in Input) error {
	var details []FieldError
	if n := utf8.RuneCountInString(strings.TrimSpace(in.Name)); n == 0 || n > 100 {
		details = append(details, FieldError{"name", "name must contain 1 to 100 characters"})
	}
	if utf8.RuneCountInString(in.Notes) > 2000 {
		details = append(details, FieldError{"notes", "notes must not exceed 2000 characters"})
	}
	if len(in.Tags) > 20 {
		details = append(details, FieldError{"tags", "at most 20 tags are allowed"})
	}
	for _, tag := range in.Tags {
		if utf8.RuneCountInString(strings.TrimSpace(tag)) == 0 || utf8.RuneCountInString(tag) > 32 {
			details = append(details, FieldError{"tags", "tags must contain 1 to 32 characters"})
			break
		}
	}
	if in.StartURL != "" {
		u, err := url.ParseRequestURI(in.StartURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			details = append(details, FieldError{"startURL", "start URL must be an HTTP(S) URL without credentials"})
		}
	}
	validBrowser := map[string]bool{"auto": true, "chrome": true, "edge": true, "brave": true, "chromium": true, "custom": true}
	if !validBrowser[in.Browser.Kind] {
		details = append(details, FieldError{"browser.kind", "unsupported browser kind"})
	}
	if in.Browser.Kind == "custom" && strings.TrimSpace(in.Browser.CustomPath) == "" {
		details = append(details, FieldError{"browser.customPath", "custom browser path is required"})
	}
	if in.Browser.Kind != "custom" && in.Browser.CustomPath != "" {
		details = append(details, FieldError{"browser.customPath", "custom path is only allowed for a custom browser"})
	}
	if report := fingerprint.Evaluate(in.Fingerprint, in.Proxy); report.HasErrors() {
		for _, issue := range report.Issues {
			if issue.Severity == fingerprint.SeverityError {
				details = append(details, FieldError{issue.Field, issue.Message})
			}
		}
	}
	if len(details) > 0 {
		return &ValidationError{Details: details}
	}
	return nil
}

func New(id string, in Input, now time.Time) (Profile, error) {
	if !ValidID(id) {
		return Profile{}, &ValidationError{Details: []FieldError{{"id", "profile ID is invalid"}}}
	}
	if err := ValidateInput(in); err != nil {
		return Profile{}, err
	}
	return Profile{ID: id, Name: strings.TrimSpace(in.Name), Notes: in.Notes,
		Tags: cleanTags(in.Tags), StartURL: in.StartURL, Browser: in.Browser,
		Fingerprint: in.Fingerprint, Proxy: in.Proxy, Revision: 1,
		CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
}

func (p Profile) Update(in Input, now time.Time) (Profile, error) {
	if err := ValidateInput(in); err != nil {
		return Profile{}, err
	}
	p.Name, p.Notes, p.Tags, p.StartURL = strings.TrimSpace(in.Name), in.Notes, cleanTags(in.Tags), in.StartURL
	p.Browser, p.Fingerprint, p.Proxy = in.Browser, in.Fingerprint, in.Proxy
	p.Revision++
	p.UpdatedAt = now.UTC()
	return p, nil
}

func (p Profile) Input() Input {
	return Input{p.Name, p.Notes, append([]string(nil), p.Tags...), p.StartURL, p.Browser, p.Fingerprint, p.Proxy}
}

func cleanTags(tags []string) []string {
	seen, out := make(map[string]bool), make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if !seen[key] {
			seen[key] = true
			out = append(out, tag)
		}
	}
	return out
}
