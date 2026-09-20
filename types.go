package hushhush

import "github.com/alrayyes/hush-hush-go/v2/internal/genclient"

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
	// AuthStatus is returned by Client.AuthStatus.
	AuthStatus = genclient.AuthStatus
	// AuditLogEntry is one entry returned by QueryAuditLog.
	AuditLogEntry = genclient.AuditLogEntry
	// AuditLogEntryAction is an AuditLogEntry's recorded action — see the
	// Action* constants below.
	AuditLogEntryAction = genclient.AuditLogEntryAction
	// AuditLogEntryActorType is an AuditLogEntry's verified actor kind —
	// see the ActorType* constants below.
	AuditLogEntryActorType = genclient.AuditLogEntryActorType
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

// Audit log entry actor types, for comparing against AuditLogEntry.ActorType.
const (
	ActorTypeSession = genclient.Session
	ActorTypeToken   = genclient.Token
)
