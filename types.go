package hushhush

import "github.com/alrayyes/hush-hush-go/internal/genclient"

// These are aliased from the generated package so a caller never needs to
// import internal/genclient directly — it's internal precisely so nothing
// outside this module depends on its shape. See CONTRIBUTING.md's "How it
// fits together".
type (
	// ObjectMetadata is returned by CreateObject, UpdateObject, and
	// GetObjectUsedBy.
	ObjectMetadata = genclient.ObjectMetadata
	// CreateObjectRequest is the payload for CreateObject.
	CreateObjectRequest = genclient.CreateObjectRequest
	// UpdateObjectRequest is the payload for UpdateObject.
	UpdateObjectRequest = genclient.UpdateObjectRequest
	// UsedBy is returned by GetObjectUsedBy.
	UsedBy = genclient.UsedBy
	// Health is returned by Client.Health.
	Health = genclient.Health
	// AuditLogEntry is one entry returned by QueryAuditLog.
	AuditLogEntry = genclient.AuditLogEntry
	// AuditLogEntryAction is an AuditLogEntry's recorded action — see the
	// Action* constants below.
	AuditLogEntryAction = genclient.AuditLogEntryAction
	// Error is hush-hush's JSON error body shape, also embedded in APIError.
	Error = genclient.Error
)

// Audit log entry actions, for comparing against AuditLogEntry.Action.
const (
	ActionCreate = genclient.Create
	ActionRead   = genclient.Read
	ActionUpdate = genclient.Update
	ActionDelete = genclient.Delete
)
