package hushhush

import "github.com/alrayyes/hush-hush-go/v4/internal/genclient"

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
	// ConsumerEntry is one directory entry returned by ListConsumers (its
	// paginated ConsumersResult.Page shape), AddConsumer, and
	// UpdateConsumer.
	ConsumerEntry = genclient.ConsumerEntry
	// ConsumersPage is ListConsumers's paginated response shape, returned
	// via ConsumersResult.Page when its filter has any field set.
	ConsumersPage = genclient.ConsumersPage
	// AddConsumerRequest is the payload for AddConsumer.
	AddConsumerRequest = genclient.AddConsumerRequest
	// UpdateConsumerRequest is the payload for UpdateConsumer.
	UpdateConsumerRequest = genclient.UpdateConsumerRequest
	// ConsumerTokenMetadata is one issued consumer token's metadata,
	// returned by ListConsumerTokens and embedded in ConsumerTokenWithValue
	// — never the raw token value, which no longer exists anywhere to
	// return once a token is created.
	ConsumerTokenMetadata = genclient.ConsumerTokenMetadata
	// ConsumerTokenWithValue is returned by CreateConsumerToken and
	// RotateConsumerToken — the only two calls that ever see the raw token
	// value.
	ConsumerTokenWithValue = genclient.ConsumerTokenWithValue
	// CreateConsumerTokenRequest is the payload for CreateConsumerToken.
	CreateConsumerTokenRequest = genclient.CreateConsumerTokenRequest
	// RotateConsumerTokenRequest is the payload for RotateConsumerToken.
	RotateConsumerTokenRequest = genclient.RotateConsumerTokenRequest
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
