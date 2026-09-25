package domain

// VisitNotification is a real-time visit event.
// Both the service (producer) and notifier (consumer) agree on this shape.
type VisitNotification struct {
	SessionID  string
	IP         string
	Country    string
	City       string
	Page       string
	PageTitle  string
	Referrer   string
	Language   string
	Timezone   string
	Viewport   string
	Device     string
	UserAgent  string
	OccurredAt int64
}
