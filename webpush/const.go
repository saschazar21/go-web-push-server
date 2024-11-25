package webpush

const (
	POSTGRES_CONNECTION_STRING_ENV = "POSTGRES_CONNECTION_STRING"

	SKIP_PADDING_ENV = "SKIP_PADDING"

	VAPID_EXPIRY_DURATION_ENV = "VAPID_EXPIRY_DURATION"
	VAPID_PRIVATE_KEY_ENV     = "VAPID_PRIVATE_KEY"
	VAPID_SUBJECT_ENV         = "VAPID_SUBJECT"
)

const (
	APPLICATION_JSON = "application/json"
	JSON_API         = "application/vnd.api+json"
	TEXT_PLAIN       = "text/plain"
)

const (
	URGENCY_VERY_LOW = "very-low"
	URGENCY_LOW      = "low"
	URGENCY_NORMAL   = "normal"
	URGENCY_HIGH     = "high"
)
