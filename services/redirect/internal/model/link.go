// services/redirect/internal/model/link.go
package model

import "time"

// LinkStatus mirrors the CockroachDB link_status enum.
type LinkStatus string

const (
	StatusActive            LinkStatus = "active"
	StatusExpired           LinkStatus = "expired"
	StatusDisabled          LinkStatus = "disabled"
	StatusFlaggedMalicious  LinkStatus = "flagged_malicious"
	StatusPasswordProtected LinkStatus = "password_protected"
)

// Link is the minimal projection the redirect hot path needs. It deliberately excludes any
// PII and the password hash (the hash lives in link_passwords and is fetched only during the
// unlock flow).
type Link struct {
	DomainID            int64      `json:"d"`
	Code                string     `json:"c"`
	WorkspaceID         int64      `json:"w"`
	Destination         string     `json:"u"`
	Status              LinkStatus `json:"s"`
	RedirectType        int16      `json:"rt"`
	Verdict             string     `json:"v"`
	IsPasswordProtected bool       `json:"pw"`
	ExpiresAt           *time.Time `json:"exp,omitempty"`
	ClickLimit          *int64     `json:"cl,omitempty"`
	ClickCount          int64      `json:"cc"`
}

// EffectiveStatus folds dynamic expiry (date / click limit) into the stored status so the
// handler has a single switch to act on.
func (l Link) EffectiveStatus(now time.Time) LinkStatus {
	if l.Status != StatusActive && l.Status != StatusPasswordProtected {
		return l.Status
	}
	if l.ExpiresAt != nil && now.After(*l.ExpiresAt) {
		return StatusExpired
	}
	if l.ClickLimit != nil && l.ClickCount >= *l.ClickLimit {
		return StatusExpired
	}
	return l.Status
}
