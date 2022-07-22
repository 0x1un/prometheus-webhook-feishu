package model

type WebhookMessage struct {
	// reference: https://prometheus.io/docs/alerting/latest/notifications/
	AlertMessage
	OpenIDs        []string
	Meta           KV
	FiringAlerts   Alerts
	ResolvedAlerts Alerts
	TitlePrefix    string
	FiringNum      int
	Severity       string
	SendNotify     string
	AlertHosts     map[string]string
}
