package notifications

type NotificationPayload struct {
	Id          string      `json:"id"`
	Bundle      string      `json:"bundle"`
	Application string      `json:"application"`
	EventType   string      `json:"event_type"`
	Timestamp   string      `json:"timestamp"`
	OrgId       string      `json:"org_id"`
	Context     struct{}    `json:"context"`
	Events      []Event     `json:"events"`
	Recipients  []Recipient `json:"recipients"`
}

type Event struct {
	Metadata struct{} `json:"metadata"`
	Payload  Payload  `json:"payload"`
}

type Recipient struct {
	Users                 []string `json:"users"`
	Emails                []string `json:"emails"`
	IgnoreUserPreferences bool     `json:"ignore_user_preferences"`
	OnlyAdmins            bool     `json:"only_admins"`
}

type Payload struct {
	Subject    string            `json:"subject"`
	GlobalVars map[string]string `json:"global_vars"`
}
