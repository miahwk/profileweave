package domain

import "time"

// TrashedProfile is a recoverable tombstone. DataRestoreToken is an opaque
// infrastructure reference and must not be returned by public interfaces.
type TrashedProfile struct {
	Profile          Profile
	DeletedAt        time.Time
	DataRestoreToken string
}
