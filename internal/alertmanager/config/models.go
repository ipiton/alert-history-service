package config

import (
	"encoding/json"
	"time"
)

// ================================================================================
// Alertmanager Configuration Models
// ================================================================================
// Complete Alertmanager configuration structure for validation (TN-151).
//
// These models represent the full Alertmanager v0.25+ configuration format.
// Compatible with: Alertmanager v0.25, v0.26, v0.27+
//
// Reference: https://prometheus.io/docs/alerting/latest/configuration/
//
// Performance Target: Unmarshal < 10ms for typical configs
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-22

// AlertmanagerConfig represents complete Alertmanager configuration.
//
// Example YAML:
//
//	global:
//	  resolve_timeout: 5m
//	route:
//	  receiver: default
//	  group_by: [alertname]
//	receivers:
//	  - name: default
//	    webhook_configs:
//	      - url: https://example.com/hook
//	inhibit_rules:
//	  - source_matchers: [alertname="Critical"]
//	    target_matchers: [alertname="Warning"]
//	    equal: [instance]
type AlertmanagerConfig struct {
	// Global contains global configuration settings
	Global *GlobalConfig `yaml:"global,omitempty" json:"global,omitempty"`

	// Route is the root route in the routing tree
	Route *Route `yaml:"route" json:"route" validate:"required"`

	// Receivers is the list of notification receivers
	Receivers []Receiver `yaml:"receivers" json:"receivers" validate:"required,min=1,dive"`

	// InhibitRules is the list of inhibition rules
	InhibitRules []InhibitRule `yaml:"inhibit_rules,omitempty" json:"inhibit_rules,omitempty" validate:"dive"`

	// MuteTimeIntervals defines time intervals for muting
	MuteTimeIntervals []MuteTimeInterval `yaml:"mute_time_intervals,omitempty" json:"mute_time_intervals,omitempty"`

	// TimeIntervals defines named time intervals (alias for MuteTimeIntervals)
	TimeIntervals []MuteTimeInterval `yaml:"time_intervals,omitempty" json:"time_intervals,omitempty"`

	// Templates is the list of template files to load
	Templates []string `yaml:"templates,omitempty" json:"templates,omitempty"`
}

// GlobalConfig represents global Alertmanager settings.
type GlobalConfig struct {
	// ResolveTimeout is the default time to wait before resolving an alert
	// Default: 5m
	ResolveTimeout Duration `yaml:"resolve_timeout,omitempty" json:"resolve_timeout,omitempty"`

	// HTTPConfig specifies default HTTP client settings
	HTTPConfig *HTTPConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	// SMTPFrom is the default SMTP From address
	SMTPFrom string `yaml:"smtp_from,omitempty" json:"smtp_from,omitempty" validate:"omitempty,email"`

	// SMTPSmartHost is the SMTP server address (host:port)
	SMTPSmartHost string `yaml:"smtp_smarthost,omitempty" json:"smtp_smarthost,omitempty"`

	// SMTPAuthUsername is SMTP authentication username
	SMTPAuthUsername string `yaml:"smtp_auth_username,omitempty" json:"smtp_auth_username,omitempty"`

	// SMTPAuthPassword is SMTP authentication password
	SMTPAuthPassword string `yaml:"smtp_auth_password,omitempty" json:"smtp_auth_password,omitempty"`

	// SMTPAuthSecret is SMTP authentication secret
	SMTPAuthSecret string `yaml:"smtp_auth_secret,omitempty" json:"smtp_auth_secret,omitempty"`

	// SMTPAuthIdentity is SMTP authentication identity
	SMTPAuthIdentity string `yaml:"smtp_auth_identity,omitempty" json:"smtp_auth_identity,omitempty"`

	// SMTPRequireTLS requires TLS for SMTP
	SMTPRequireTLS *bool `yaml:"smtp_require_tls,omitempty" json:"smtp_require_tls,omitempty"`

	// SlackAPIURL is the default Slack webhook URL
	SlackAPIURL string `yaml:"slack_api_url,omitempty" json:"slack_api_url,omitempty" validate:"omitempty,url"`

	// SlackAPIURLFile is path to file containing Slack webhook URL
	SlackAPIURLFile string `yaml:"slack_api_url_file,omitempty" json:"slack_api_url_file,omitempty"`

	// PagerdutyURL is the PagerDuty API URL
	PagerdutyURL string `yaml:"pagerduty_url,omitempty" json:"pagerduty_url,omitempty" validate:"omitempty,url"`

	// OpsGenieAPIURL is the OpsGenie API URL
	OpsGenieAPIURL string `yaml:"opsgenie_api_url,omitempty" json:"opsgenie_api_url,omitempty" validate:"omitempty,url"`

	// OpsGenieAPIKey is the OpsGenie API key
	OpsGenieAPIKey string `yaml:"opsgenie_api_key,omitempty" json:"opsgenie_api_key,omitempty"`

	// OpsGenieAPIKeyFile is path to file containing OpsGenie API key
	OpsGenieAPIKeyFile string `yaml:"opsgenie_api_key_file,omitempty" json:"opsgenie_api_key_file,omitempty"`
}

// Route represents a routing node in the routing tree.
type Route struct {
	// Receiver is the name of the receiver to send notifications to
	Receiver string `yaml:"receiver,omitempty" json:"receiver,omitempty"`

	// GroupBy is the list of labels to group alerts by
	GroupBy []string `yaml:"group_by,omitempty" json:"group_by,omitempty"`

	// GroupWait is how long to wait before sending initial notification
	GroupWait *Duration `yaml:"group_wait,omitempty" json:"group_wait,omitempty"`

	// GroupInterval is how long to wait before sending batch of new alerts
	GroupInterval *Duration `yaml:"group_interval,omitempty" json:"group_interval,omitempty"`

	// RepeatInterval is how long to wait before resending notification
	RepeatInterval *Duration `yaml:"repeat_interval,omitempty" json:"repeat_interval,omitempty"`

	// Matchers is the list of matchers (new format: label=value)
	Matchers []string `yaml:"matchers,omitempty" json:"matchers,omitempty"`

	// Match is deprecated label matching (exact match)
	Match map[string]string `yaml:"match,omitempty" json:"match,omitempty"`

	// MatchRE is deprecated label matching (regex match)
	MatchRE map[string]string `yaml:"match_re,omitempty" json:"match_re,omitempty"`

	// Continue determines if processing continues to sibling routes
	Continue bool `yaml:"continue,omitempty" json:"continue,omitempty"`

	// Routes is the list of child routes
	Routes []Route `yaml:"routes,omitempty" json:"routes,omitempty"`

	// MuteTimeIntervals is the list of mute time intervals
	MuteTimeIntervals []string `yaml:"mute_time_intervals,omitempty" json:"mute_time_intervals,omitempty"`

	// ActiveTimeIntervals is the list of active time intervals
	ActiveTimeIntervals []string `yaml:"active_time_intervals,omitempty" json:"active_time_intervals,omitempty"`
}

// Receiver represents a notification receiver.
type Receiver struct {
	// Name is the unique receiver name
	Name string `yaml:"name" json:"name" validate:"required"`

	// EmailConfigs is the list of email notification configurations
	EmailConfigs []EmailConfig `yaml:"email_configs,omitempty" json:"email_configs,omitempty"`

	// PagerdutyConfigs is the list of PagerDuty notification configurations
	PagerdutyConfigs []PagerdutyConfig `yaml:"pagerduty_configs,omitempty" json:"pagerduty_configs,omitempty"`

	// SlackConfigs is the list of Slack notification configurations
	SlackConfigs []SlackConfig `yaml:"slack_configs,omitempty" json:"slack_configs,omitempty"`

	// WebhookConfigs is the list of webhook notification configurations
	WebhookConfigs []WebhookConfig `yaml:"webhook_configs,omitempty" json:"webhook_configs,omitempty"`

	// OpsGenieConfigs is the list of OpsGenie notification configurations
	OpsGenieConfigs []OpsGenieConfig `yaml:"opsgenie_configs,omitempty" json:"opsgenie_configs,omitempty"`

	// VictorOpsConfigs is the list of VictorOps notification configurations
	VictorOpsConfigs []VictorOpsConfig `yaml:"victorops_configs,omitempty" json:"victorops_configs,omitempty"`

	// PushoverConfigs is the list of Pushover notification configurations
	PushoverConfigs []PushoverConfig `yaml:"pushover_configs,omitempty" json:"pushover_configs,omitempty"`

	// WeChatConfigs is the list of WeChat notification configurations
	WeChatConfigs []WeChatConfig `yaml:"wechat_configs,omitempty" json:"wechat_configs,omitempty"`
}

// HasAnyIntegration returns true if the receiver has at least one integration configured.
func (r *Receiver) HasAnyIntegration() bool {
	return len(r.EmailConfigs) > 0 ||
		len(r.PagerdutyConfigs) > 0 ||
		len(r.SlackConfigs) > 0 ||
		len(r.WebhookConfigs) > 0 ||
		len(r.OpsGenieConfigs) > 0 ||
		len(r.VictorOpsConfigs) > 0 ||
		len(r.PushoverConfigs) > 0 ||
		len(r.WeChatConfigs) > 0
}

// EmailConfig represents email notification configuration.
type EmailConfig struct {
	SendResolved *bool      `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	To           string     `yaml:"to" json:"to" validate:"required,email"`
	From         string     `yaml:"from,omitempty" json:"from,omitempty" validate:"omitempty,email"`
	Smarthost    string     `yaml:"smarthost,omitempty" json:"smarthost,omitempty"`
	AuthUsername string     `yaml:"auth_username,omitempty" json:"auth_username,omitempty"`
	AuthPassword string     `yaml:"auth_password,omitempty" json:"auth_password,omitempty"`
	AuthSecret   string     `yaml:"auth_secret,omitempty" json:"auth_secret,omitempty"`
	AuthIdentity string     `yaml:"auth_identity,omitempty" json:"auth_identity,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	HTML         string     `yaml:"html,omitempty" json:"html,omitempty"`
	Text         string     `yaml:"text,omitempty" json:"text,omitempty"`
	RequireTLS   *bool      `yaml:"require_tls,omitempty" json:"require_tls,omitempty"`
	TLSConfig    *TLSConfig `yaml:"tls_config,omitempty" json:"tls_config,omitempty"`
}

// PagerdutyConfig represents PagerDuty notification configuration.
type PagerdutyConfig struct {
	SendResolved *bool             `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	RoutingKey   string            `yaml:"routing_key,omitempty" json:"routing_key,omitempty"`
	ServiceKey   string            `yaml:"service_key,omitempty" json:"service_key,omitempty"`
	URL          string            `yaml:"url,omitempty" json:"url,omitempty" validate:"omitempty,url"`
	Client       string            `yaml:"client,omitempty" json:"client,omitempty"`
	ClientURL    string            `yaml:"client_url,omitempty" json:"client_url,omitempty" validate:"omitempty,url"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Severity     string            `yaml:"severity,omitempty" json:"severity,omitempty"`
	Details      map[string]string `yaml:"details,omitempty" json:"details,omitempty"`
	HTTPConfig   *HTTPConfig       `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// SlackConfig represents Slack notification configuration.
type SlackConfig struct {
	SendResolved *bool       `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	APIURL       string      `yaml:"api_url,omitempty" json:"api_url,omitempty" validate:"omitempty,url"`
	APIURLFile   string      `yaml:"api_url_file,omitempty" json:"api_url_file,omitempty"`
	Channel      string      `yaml:"channel,omitempty" json:"channel,omitempty"`
	Username     string      `yaml:"username,omitempty" json:"username,omitempty"`
	Color        string      `yaml:"color,omitempty" json:"color,omitempty"`
	Title        string      `yaml:"title,omitempty" json:"title,omitempty"`
	TitleLink    string      `yaml:"title_link,omitempty" json:"title_link,omitempty" validate:"omitempty,url"`
	Pretext      string      `yaml:"pretext,omitempty" json:"pretext,omitempty"`
	Text         string      `yaml:"text,omitempty" json:"text,omitempty"`
	Fields       []SlackField `yaml:"fields,omitempty" json:"fields,omitempty"`
	ShortFields  bool        `yaml:"short_fields,omitempty" json:"short_fields,omitempty"`
	Footer       string      `yaml:"footer,omitempty" json:"footer,omitempty"`
	Fallback     string      `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	CallbackID   string      `yaml:"callback_id,omitempty" json:"callback_id,omitempty"`
	IconEmoji    string      `yaml:"icon_emoji,omitempty" json:"icon_emoji,omitempty"`
	IconURL      string      `yaml:"icon_url,omitempty" json:"icon_url,omitempty" validate:"omitempty,url"`
	ImageURL     string      `yaml:"image_url,omitempty" json:"image_url,omitempty" validate:"omitempty,url"`
	ThumbURL     string      `yaml:"thumb_url,omitempty" json:"thumb_url,omitempty" validate:"omitempty,url"`
	LinkNames    bool        `yaml:"link_names,omitempty" json:"link_names,omitempty"`
	MrkdwnIn     []string    `yaml:"mrkdwn_in,omitempty" json:"mrkdwn_in,omitempty"`
	HTTPConfig   *HTTPConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// SlackField represents a Slack message field.
type SlackField struct {
	Title string `yaml:"title" json:"title"`
	Value string `yaml:"value" json:"value"`
	Short *bool  `yaml:"short,omitempty" json:"short,omitempty"`
}

// WebhookConfig represents webhook notification configuration.
type WebhookConfig struct {
	SendResolved *bool       `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	URL          string      `yaml:"url" json:"url" validate:"required,url"`
	HTTPConfig   *HTTPConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`
	MaxAlerts    int         `yaml:"max_alerts,omitempty" json:"max_alerts,omitempty" validate:"omitempty,min=0"`
}

// OpsGenieConfig represents OpsGenie notification configuration.
type OpsGenieConfig struct {
	SendResolved *bool             `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	APIKey       string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	APIKeyFile   string            `yaml:"api_key_file,omitempty" json:"api_key_file,omitempty"`
	APIURL       string            `yaml:"api_url,omitempty" json:"api_url,omitempty" validate:"omitempty,url"`
	Message      string            `yaml:"message,omitempty" json:"message,omitempty"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Source       string            `yaml:"source,omitempty" json:"source,omitempty"`
	Details      map[string]string `yaml:"details,omitempty" json:"details,omitempty"`
	Responders   []OpsGenieResponder `yaml:"responders,omitempty" json:"responders,omitempty"`
	Tags         []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Note         string            `yaml:"note,omitempty" json:"note,omitempty"`
	Priority     string            `yaml:"priority,omitempty" json:"priority,omitempty"`
	HTTPConfig   *HTTPConfig       `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// OpsGenieResponder represents an OpsGenie responder.
type OpsGenieResponder struct {
	ID       string `yaml:"id,omitempty" json:"id,omitempty"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Type     string `yaml:"type" json:"type" validate:"required,oneof=team user escalation schedule"`
}

// VictorOpsConfig represents VictorOps notification configuration.
type VictorOpsConfig struct {
	SendResolved  *bool             `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	APIKey        string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	APIKeyFile    string            `yaml:"api_key_file,omitempty" json:"api_key_file,omitempty"`
	APIURL        string            `yaml:"api_url,omitempty" json:"api_url,omitempty" validate:"omitempty,url"`
	RoutingKey    string            `yaml:"routing_key,omitempty" json:"routing_key,omitempty"`
	MessageType   string            `yaml:"message_type,omitempty" json:"message_type,omitempty"`
	EntityDisplayName string        `yaml:"entity_display_name,omitempty" json:"entity_display_name,omitempty"`
	StateMessage  string            `yaml:"state_message,omitempty" json:"state_message,omitempty"`
	MonitoringTool string           `yaml:"monitoring_tool,omitempty" json:"monitoring_tool,omitempty"`
	CustomFields  map[string]string `yaml:"custom_fields,omitempty" json:"custom_fields,omitempty"`
	HTTPConfig    *HTTPConfig       `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// PushoverConfig represents Pushover notification configuration.
type PushoverConfig struct {
	SendResolved *bool       `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	UserKey      string      `yaml:"user_key,omitempty" json:"user_key,omitempty"`
	UserKeyFile  string      `yaml:"user_key_file,omitempty" json:"user_key_file,omitempty"`
	Token        string      `yaml:"token,omitempty" json:"token,omitempty"`
	TokenFile    string      `yaml:"token_file,omitempty" json:"token_file,omitempty"`
	Title        string      `yaml:"title,omitempty" json:"title,omitempty"`
	Message      string      `yaml:"message,omitempty" json:"message,omitempty"`
	URL          string      `yaml:"url,omitempty" json:"url,omitempty" validate:"omitempty,url"`
	URLTitle     string      `yaml:"url_title,omitempty" json:"url_title,omitempty"`
	Sound        string      `yaml:"sound,omitempty" json:"sound,omitempty"`
	Priority     string      `yaml:"priority,omitempty" json:"priority,omitempty"`
	Retry        *Duration   `yaml:"retry,omitempty" json:"retry,omitempty"`
	Expire       *Duration   `yaml:"expire,omitempty" json:"expire,omitempty"`
	HTML         bool        `yaml:"html,omitempty" json:"html,omitempty"`
	HTTPConfig   *HTTPConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// WeChatConfig represents WeChat notification configuration.
type WeChatConfig struct {
	SendResolved *bool       `yaml:"send_resolved,omitempty" json:"send_resolved,omitempty"`
	APIURL       string      `yaml:"api_url,omitempty" json:"api_url,omitempty" validate:"omitempty,url"`
	APISecret    string      `yaml:"api_secret,omitempty" json:"api_secret,omitempty"`
	CorpID       string      `yaml:"corp_id,omitempty" json:"corp_id,omitempty"`
	AgentID      string      `yaml:"agent_id,omitempty" json:"agent_id,omitempty"`
	ToUser       string      `yaml:"to_user,omitempty" json:"to_user,omitempty"`
	ToParty      string      `yaml:"to_party,omitempty" json:"to_party,omitempty"`
	ToTag        string      `yaml:"to_tag,omitempty" json:"to_tag,omitempty"`
	Message      string      `yaml:"message,omitempty" json:"message,omitempty"`
	MessageType  string      `yaml:"message_type,omitempty" json:"message_type,omitempty"`
	HTTPConfig   *HTTPConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// HTTPConfig represents HTTP client configuration.
type HTTPConfig struct {
	ProxyURL            string      `yaml:"proxy_url,omitempty" json:"proxy_url,omitempty" validate:"omitempty,url"`
	TLSConfig           *TLSConfig  `yaml:"tls_config,omitempty" json:"tls_config,omitempty"`
	BearerToken         string      `yaml:"bearer_token,omitempty" json:"bearer_token,omitempty"`
	BearerTokenFile     string      `yaml:"bearer_token_file,omitempty" json:"bearer_token_file,omitempty"`
	BasicAuth           *BasicAuth  `yaml:"basic_auth,omitempty" json:"basic_auth,omitempty"`
	FollowRedirects     *bool       `yaml:"follow_redirects,omitempty" json:"follow_redirects,omitempty"`
	EnableHTTP2         *bool       `yaml:"enable_http2,omitempty" json:"enable_http2,omitempty"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	CAFile             string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	ServerName         string `yaml:"server_name,omitempty" json:"server_name,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
}

// BasicAuth represents HTTP basic authentication.
type BasicAuth struct {
	Username     string `yaml:"username,omitempty" json:"username,omitempty"`
	Password     string `yaml:"password,omitempty" json:"password,omitempty"`
	PasswordFile string `yaml:"password_file,omitempty" json:"password_file,omitempty"`
}

// InhibitRule represents an inhibition rule.
type InhibitRule struct {
	// SourceMatchers are matchers for source alerts (new format)
	SourceMatchers []string `yaml:"source_matchers,omitempty" json:"source_matchers,omitempty"`

	// TargetMatchers are matchers for target alerts (new format)
	TargetMatchers []string `yaml:"target_matchers,omitempty" json:"target_matchers,omitempty"`

	// SourceMatch is deprecated source label matching (exact)
	SourceMatch map[string]string `yaml:"source_match,omitempty" json:"source_match,omitempty"`

	// SourceMatchRE is deprecated source label matching (regex)
	SourceMatchRE map[string]string `yaml:"source_match_re,omitempty" json:"source_match_re,omitempty"`

	// TargetMatch is deprecated target label matching (exact)
	TargetMatch map[string]string `yaml:"target_match,omitempty" json:"target_match,omitempty"`

	// TargetMatchRE is deprecated target label matching (regex)
	TargetMatchRE map[string]string `yaml:"target_match_re,omitempty" json:"target_match_re,omitempty"`

	// Equal is the list of labels that must be equal for inhibition
	Equal []string `yaml:"equal,omitempty" json:"equal,omitempty"`
}

// MuteTimeInterval defines a named time interval for muting.
type MuteTimeInterval struct {
	Name          string         `yaml:"name" json:"name" validate:"required"`
	TimeIntervals []TimeInterval `yaml:"time_intervals" json:"time_intervals" validate:"required,min=1"`
}

// TimeInterval represents a time interval.
type TimeInterval struct {
	Times       []TimeRange `yaml:"times,omitempty" json:"times,omitempty"`
	Weekdays    []string    `yaml:"weekdays,omitempty" json:"weekdays,omitempty"`
	DaysOfMonth []string    `yaml:"days_of_month,omitempty" json:"days_of_month,omitempty"`
	Months      []string    `yaml:"months,omitempty" json:"months,omitempty"`
	Years       []string    `yaml:"years,omitempty" json:"years,omitempty"`
}

// TimeRange represents a time range (HH:MM - HH:MM).
type TimeRange struct {
	StartTime string `yaml:"start_time" json:"start_time" validate:"required"`
	EndTime   string `yaml:"end_time" json:"end_time" validate:"required"`
}

// Duration is a custom duration type that supports YAML/JSON unmarshaling.
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// String returns the string representation of the duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}
