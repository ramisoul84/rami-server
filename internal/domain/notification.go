package domain

// VisitKind tells a first-time visitor apart from one coming back.
type VisitKind string

const (
	VisitKindNew       VisitKind = "new"
	VisitKindReturning VisitKind = "returning"
)

// VisitNotification is sent once per session, on its first pageview.
// Both the service (producer) and notifier (consumer) agree on this shape.
type VisitNotification struct {
	Kind VisitKind
	// VisitNumber is this session's ordinal for the visitor: 1 for a new visitor.
	VisitNumber  int
	SessionID    string
	VisitorID    string
	IP           string
	Country      string
	City         string
	Page         string
	PageTitle    string
	Referrer     string
	Source       string
	Campaign     string
	Language     string
	SiteLanguage string
	Timezone     string
	Viewport     string
	Device       string
	OS           string
	Browser      string
	UserAgent    string
	OccurredAt   int64
}

// Lead event names, shared with the client's analytics-events.ts.
const (
	LeadResumeDownload = "resume_download"
	LeadContactClick   = "contact_click"
)

// LeadNotification is sent when a visitor shows hiring intent (resume download, contact click).
type LeadNotification struct {
	Name       string // LeadResumeDownload | LeadContactClick
	Detail     string // resume language or contact channel
	Origin     string // where on the site it was triggered (hero, contact, terminal…)
	SessionID  string
	Country    string
	City       string
	Device     string
	OS         string
	Browser    string
	Page       string
	Source     string
	OccurredAt int64
}

// NotifyMode is an admin's real-time alert preference.
type NotifyMode string

const (
	NotifyAll NotifyMode = "all" // every new session + leads
	NotifyNew NotifyMode = "new" // first-time visitors + leads
	NotifyOff NotifyMode = "off" // nothing real-time
)

// AdminChat is a registered Telegram admin and their preferences.
type AdminChat struct {
	Username   string     `db:"username"`
	ChatID     int64      `db:"chat_id"`
	NotifyMode NotifyMode `db:"notify_mode"`
	Digest     bool       `db:"digest"`
}
