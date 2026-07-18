package domain

import (
	"context"
	"time"
)

type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
	StatusFailed   Status = "failed"
)

type Session struct {
	ProfileID string     `json:"profileId"`
	Status    Status     `json:"status"`
	PID       int        `json:"pid,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	StoppedAt *time.Time `json:"stoppedAt,omitempty"`
	LastError string     `json:"lastError,omitempty"`
}

type BrowserDescriptor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Available bool   `json:"available"`
}

type LaunchSpec struct {
	ProfileID    string
	BrowserKind  string
	CustomPath   string
	StartURL     string
	Locale       string
	Width        int
	Height       int
	DPR          float64
	UAMode       string
	UserAgent    string
	ProxyMode    string
	ProxyHost    string
	ProxyPort    int
	WebRTCPolicy string
}

type Process struct {
	PID  int
	Done <-chan error
}

type CapabilityStatus string

const (
	CapabilityApplied     CapabilityStatus = "applied"
	CapabilityPartial     CapabilityStatus = "partial"
	CapabilityUnsupported CapabilityStatus = "unsupported"
)

// ProviderCapability states what a runtime actually applies. It describes
// observable behavior rather than promising that a browser is undetectable.
type ProviderCapability struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Status CapabilityStatus `json:"status"`
	Detail string           `json:"detail"`
}

type ProviderInfo struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	Source            string               `json:"source"`
	License           string               `json:"license"`
	VersionManagement string               `json:"versionManagement"`
	Capabilities      []ProviderCapability `json:"capabilities"`
}

type DoctorIssue struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type DoctorReport struct {
	Provider           ProviderInfo        `json:"provider"`
	Healthy            bool                `json:"healthy"`
	InspectedBrowsers  int                 `json:"inspectedBrowsers"`
	AvailableBrowsers  int                 `json:"availableBrowsers"`
	ActiveSessions     int                 `json:"activeSessions"`
	Browsers           []BrowserDescriptor `json:"browsers"`
	Issues             []DoctorIssue       `json:"issues"`
}

type Runtime interface {
	Discover(context.Context) ([]BrowserDescriptor, error)
	Launch(context.Context, LaunchSpec) (Process, error)
	Stop(context.Context, string) error
}

// Provider is a Runtime that can describe its provenance and honest feature
// boundary. Runtime remains small so test doubles and third-party adapters can
// be introduced incrementally.
type Provider interface {
	Runtime
	Info() ProviderInfo
}
