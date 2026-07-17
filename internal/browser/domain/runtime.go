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

type Runtime interface {
	Discover(context.Context) ([]BrowserDescriptor, error)
	Launch(context.Context, LaunchSpec) (Process, error)
	Stop(context.Context, string) error
}
