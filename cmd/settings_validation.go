package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
	"golang.org/x/net/idna"
)

// This bounds each manager queue to 200000 entries, independently of the
// individual controls exposed by the performance form.
const maxSettingsRateProduct = 100000

func (a *App) validateSettings(s models.Settings) error {
	if err := validateSettings(s); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate i18n.
	if _, err := a.fs.Get("/i18n/" + s.AppLang + ".json"); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "app.lang: unknown language")
	}

	return a.validateOIDCSettings(s)
}

// validateSettings sanitizes and validates all setting values.
func validateSettings(s models.Settings) error {
	var err error

	// Keep the first field error while checking each settings group.
	check := func(ok bool, field, message string) {
		if !ok && err == nil {
			err = fmt.Errorf("%s: %s", field, message)
		}
	}

	// Evaluate text fields (length, required, and disallowed chars).
	text := func(value, field string, required bool) {
		check(!required || strings.TrimSpace(value) != "", field, "is required")
		check(utf8.RuneCountInString(value) <= stdInputMaxLen, field, "is too long")
		check(!strings.ContainsAny(value, "\r\n\x00"), field, "must not contain line breaks or NUL")
	}

	// duration validates a duration string and checks whether it's in the given bounds.
	duration := func(value, field string, min, max time.Duration) {
		d, e := settingsDuration(value)
		check(e == nil && d >= min && d <= max, field,
			fmt.Sprintf("must be a duration between %s and %s", min, max))
	}

	// enum validates that the value is one of the allowed options.
	enum := func(value, field string, allowed ...string) {
		check(slices.Contains(allowed, value), field, "must be one of: "+strings.Join(allowed, ", "))
	}

	// addr validates that the value is an email address.
	addr := func(value, field string) {
		_, e := mail.ParseAddress(value)
		check(e == nil && !strings.ContainsAny(value, "\r\n\x00"), field, "must be an email address")
	}

	// cron checks if the given value is a valid cron string.
	cron := func(value, field string) {
		_, e := parseCronString(value)
		check(e == nil, field, "must be a valid cron expression")
	}

	// ============================ Validate fields.

	// Site and URLs.
	text(s.AppSiteName, "app.site_name", true)
	check(settingsURL(s.AppRootURL, false), "app.root_url", "must be an canonical URL")
	if u, e := url.Parse(s.AppRootURL); e == nil {
		check(u.RawQuery == "" && !u.ForceQuery && u.Fragment == "", "app.root_url", "must not contain a query or fragment")
	}
	check(s.AppLogoURL == "" || settingsURL(s.AppLogoURL, true), "app.logo_url", "must be a canonical URL or an absolute path")
	check(s.AppFaviconURL == "" || settingsURL(s.AppFaviconURL, true), "app.favicon_url", "must be a canonical URL or an absolute path")

	// Sender defaults and notification recipients.
	if _, e := sanitizeFromAddresses(s.AppFromEmail); e != nil {
		check(false, "app.from_email", e.Error())
	}
	for i, v := range s.AppNotifyEmails {
		addr(v, fmt.Sprintf("app.notify_emails[%d]", i))
	}

	check(s.AppLang != "" && len(s.AppLang) <= 6 && !reLangCode.MatchString(s.AppLang), "app.lang", "must be a valid language code")

	// Performance numbers.
	check(s.AppBatchSize >= 1 && s.AppBatchSize <= 100000, "app.batch_size", "must be between 1 and 100000")
	check(s.AppConcurrency >= 1 && s.AppConcurrency <= 10000, "app.concurrency", "must be between 1 and 10000")
	check(s.AppMessageRate >= 1 && s.AppMessageRate <= 100000, "app.message_rate", "must be between 1 and 100000")
	if s.AppConcurrency > 0 && s.AppMessageRate > 0 {
		check(s.AppConcurrency <= maxSettingsRateProduct/s.AppMessageRate, "app.message_rate", "concurrency * message_rate must not exceed 100000")
	}
	check(s.AppMaxSendErrors >= 0 && s.AppMaxSendErrors <= 100000, "app.max_send_errors", "must be between 0 and 100000 (0 disables the threshold)")

	// Sliding window.
	check(s.AppMessageSlidingWindowRate >= 0 && s.AppMessageSlidingWindowRate <= 10000000 && (!s.AppMessageSlidingWindow || s.AppMessageSlidingWindowRate > 0),
		"app.message_sliding_window_rate", "must be between 1 and 10000000 when enabled, otherwise 0 to 10000000")
	if s.AppMessageSlidingWindow {
		duration(s.AppMessageSlidingWindowDuration, "app.message_sliding_window_duration", time.Second+1, math.MaxInt64)
	}

	if s.CacheSlowQueries {
		cron(s.CacheSlowQueriesInterval, "app.cache_slow_queries_interval")
	}

	// Exportable subscriber data and domain restrictions.
	for i, v := range s.PrivacyExportable {
		enum(v, fmt.Sprintf("privacy.exportable[%d]", i), "profile", "subscriptions", "campaign_views", "link_clicks")
	}
	for i, v := range s.DomainBlocklist {
		check(settingsDomainPattern(v), fmt.Sprintf("privacy.domain_blocklist[%d]", i), "must be a valid domain name or wildcard domain")
	}
	for i, v := range s.DomainAllowlist {
		check(settingsDomainPattern(v), fmt.Sprintf("privacy.domain_allowlist[%d]", i), "must be a valid domain name or wildcard domain")
	}

	// Session and signed-link lifetimes.
	duration(s.SecuritySessionDuration, "security.session_duration", time.Second, math.MaxInt64)
	if s.SecuritySignedLinks.Enabled || s.SecuritySignedLinks.TTL != "" {
		if s.SecuritySignedLinks.TTL != "0" {
			duration(s.SecuritySignedLinks.TTL, "security.signed_links.ttl", time.Second, math.MaxInt64)
		}
	}

	for i, v := range s.SecurityTrustedURLs {
		check(v == "*" || settingsURL(v, false), fmt.Sprintf("security.trusted_urls[%d]", i), "must be an valid URL or *")
	}

	// Only the selected CAPTCHA provider needs credentials.
	c := s.SecurityCaptcha
	check(!c.Altcha.Enabled || !c.HCaptcha.Enabled, "security.captcha", "only one provider can be enabled")
	check(c.Altcha.Complexity >= 0 && c.Altcha.Complexity <= 10000000 && (!c.Altcha.Enabled || c.Altcha.Complexity > 0), "security.captcha.altcha.complexity", "must be between 1 and 10000000 when enabled, otherwise 0 to 10000000")
	text(c.HCaptcha.Key, "security.captcha.hcaptcha.key", c.HCaptcha.Enabled)
	text(c.HCaptcha.Secret, "security.captcha.hcaptcha.secret", c.HCaptcha.Enabled)

	// OIDC endpoints and credentials.
	o := s.OIDC
	if o.Enabled || o.ProviderURL != "" {
		check(settingsURL(o.ProviderURL, false), "security.oidc.provider_url", "must be an HTTP(S) URL")
	}
	if o.Enabled || o.RedirectURL != "" {
		check(settingsURL(o.RedirectURL, false), "security.oidc.redirect_url", "must be an HTTP(S) URL")
	}
	text(o.ProviderName, "security.oidc.provider_name", false)
	text(o.ClientID, "security.oidc.client_id", o.Enabled)
	text(o.ClientSecret, "security.oidc.client_secret", o.Enabled)

	// OIDC roles are validated against the DB by validateOIDCSettings() in the end.
	for i, role := range o.Roles {
		text(role.Claim, fmt.Sprintf("security.oidc.roles[%d].claim", i), true)
		text(role.Match, fmt.Sprintf("security.oidc.roles[%d].match", i), true)
	}

	// Upload limits.
	enum(s.UploadProvider, "upload.provider", "filesystem", "s3")
	if e := validateUploadMaxFileSize(s.UploadMaxFileSize); e != nil {
		check(false, "upload.max_file_size", "must be a positive KB value")
	}
	for i, v := range s.UploadExtensions {
		check(v == "*" || settingsFileExt(v), fmt.Sprintf("upload.extensions[%d]", i), "must be a file extension or *")
	}

	// Filesystem uploads need a directory and a public URL path.
	text(s.UploadFilesystem.UploadPath, "upload.filesystem.upload_path", s.UploadProvider == "filesystem")
	if s.UploadProvider == "filesystem" || s.UploadFilesystem.UploadURI != "" {
		check(settingsURL(s.UploadFilesystem.UploadURI, true) &&
			strings.HasPrefix(s.UploadFilesystem.UploadURI, "/") &&
			s.UploadFilesystem.UploadURI != "/",
			"upload.filesystem.upload_uri", "must be a non-root absolute URI")
		check(!strings.ContainsAny(s.UploadFilesystem.UploadURI, "?#"), "upload.filesystem.upload_uri", "must not contain a query or fragment")
	}

	// S3 endpoints, bucket details and authentication.
	u := s.UploadS3
	if u.URL != "" {
		check(settingsURL(u.URL, false), "upload.s3.url", "must be a valid URL")
	}
	if u.PublicURL != "" {
		check(settingsURL(u.PublicURL, true), "upload.s3.public_url", "must be an valid URL or an absolute path")
	}

	text(u.AwsDefaultRegion, "upload.s3.aws_default_region", s.UploadProvider == "s3")
	text(u.Bucket, "upload.s3.bucket", s.UploadProvider == "s3")
	if u.Bucket != "" {
		check(settingsDomain(strings.ReplaceAll(u.Bucket, "_", "-")), "upload.s3.bucket", "must be a bucket name without a path or whitespace")
	}
	text(u.BucketDomain, "upload.s3.bucket_domain", false)
	text(u.BucketPath, "upload.s3.bucket_path", false)

	text(u.AwsAccessKeyID, "upload.s3.aws_access_key_id", false)
	text(u.AwsSecretAccessKey, "upload.s3.aws_secret_access_key", false)
	if s.UploadProvider == "s3" {
		// Both empty selects IAM authentication
		check((u.AwsAccessKeyID == "") == (u.AwsSecretAccessKey == ""),
			"upload.s3.aws_secret_access_key", "access key and secret must be either present together or both both should be empty to use IAM")
	}

	if s.UploadProvider == "s3" || u.BucketType != "" {
		enum(u.BucketType, "upload.s3.bucket_type", "public", "private")
	}
	if s.UploadProvider == "s3" {
		// Zero selects the provider's default expiry.
		duration(u.Expiry, "upload.s3.expiry", 0, 7*24*time.Hour)
	}

	// UUIDs identify preserved passwords, including in disabled entries.
	seen := make(map[string]bool)
	uuid := func(value, field string) {
		if value == "" {
			return
		}

		id, e := uuid.FromString(value)
		value = id.String()

		check(e == nil && !seen[value], field, "must be a unique UUID")
		seen[value] = true
	}

	// SMTP.
	for i, v := range s.SMTP {
		p := fmt.Sprintf("smtp[%d].", i)
		uuid(v.UUID, p+"uuid")
		text(v.Name, p+"name", v.ExcludeFromDefault)
		text(v.Username, p+"username", v.Enabled && v.AuthProtocol != "" && v.AuthProtocol != "none")
		text(v.Password, p+"password", false)

		if v.Enabled || v.Host != "" {
			check(settingsHost(v.Host), p+"host", "must be a hostname or IP address without a port")
		}
		if v.Enabled || v.Port != 0 {
			check(v.Port >= 1 && v.Port <= 65535, p+"port", "must be between 1 and 65535")
		}

		check(v.HelloHostname == "" || settingsHost(v.HelloHostname), p+"hello_hostname", "must be a hostname or an IP address")
		enum(v.AuthProtocol, p+"auth_protocol", "", "none", "plain", "login", "cram")
		if v.Enabled || v.TLSType != "" {
			enum(v.TLSType, p+"tls_type", "none", "TLS", "STARTTLS")
		}

		check(v.MaxConns >= 0 && v.MaxConns <= 1000 && (!v.Enabled || v.MaxConns > 0),
			p+"max_conns", "must be between 1 and 1000 when enabled, otherwise 0 to 1000")
		check(v.MaxMsgRetries >= 0 && v.MaxMsgRetries <= 1000, p+"max_msg_retries", "must be between 0 and 1000")
		if v.MsgRetryDelay != "" {
			duration(v.MsgRetryDelay, p+"msg_retry_delay", 0, math.MaxInt64)
		}
		if v.Enabled || v.IdleTimeout != "" {
			duration(v.IdleTimeout, p+"idle_timeout", 0, math.MaxInt64)
		}
		if v.Enabled || v.WaitTimeout != "" {
			duration(v.WaitTimeout, p+"wait_timeout", 0, math.MaxInt64)
		}

		for j, headers := range v.EmailHeaders {
			for name, value := range headers {
				check(settingsHTTPHdr(name) && !strings.ContainsAny(value, "\r\n\x00"),
					fmt.Sprintf("%semail_headers[%d]", p, j), "must contain valid header names and values without line breaks or NUL")
			}
		}

		for j, value := range v.FromAddresses {
			if strings.Contains(value, "@") {
				a, e := mail.ParseAddress(value)
				check(e == nil && a.Address == value, fmt.Sprintf("%sfrom_addresses[%d]", p, j), "must be a bare email address without a display name")
			} else {
				check(settingsDomain(value), fmt.Sprintf("%sfrom_addresses[%d]", p, j), "must be a valid email address or domain")
			}
		}
	}

	// HTTP messengers.
	clear(seen)
	for i, v := range s.Messengers {
		p := fmt.Sprintf("messengers[%d].", i)
		uuid(v.UUID, p+"uuid")
		text(v.Name, p+"name", true)
		text(v.Username, p+"username", false)
		text(v.Password, p+"password", false)

		if v.Enabled || v.RootURL != "" {
			check(settingsURL(v.RootURL, false), p+"root_url", "must be an HTTP(S) URL")
		}

		// net/http accepts zero for unlimited connections and no timeout.
		check(v.MaxConns >= 0 && v.MaxConns <= 1000, p+"max_conns", "must be between 0 and 1000")
		check(v.MaxMsgRetries >= 0 && v.MaxMsgRetries <= 1000, p+"max_msg_retries", "must be between 0 and 1000")
		if v.Enabled || v.Timeout != "" {
			duration(v.Timeout, p+"timeout", 0, math.MaxInt64)
		}
	}

	// Bounces.
	for name := range s.BounceActions {
		enum(name, "bounce.actions", "soft", "hard", "complaint")
	}
	for _, name := range []string{"soft", "hard", "complaint"} {
		v, ok := s.BounceActions[name]
		check(ok, "bounce.actions."+name, "is required")
		check(v.Count > 0, "bounce.actions."+name+".count", "must be positive")
		enum(v.Action, "bounce.actions."+name+".action", "none", "unsubscribe", "blocklist", "delete")
	}

	// Bounce webhooks.
	webhooks := s.BounceEnabled && s.BounceEnableWebhooks
	text(s.BounceSendgrid.Key, "bounce.sendgrid.key", webhooks && s.BounceSendgrid.Enabled)
	if s.BounceSendgrid.Key != "" {
		// The webhook constructor expects an ECDSA public key.
		b, e := base64.StdEncoding.DecodeString(s.BounceSendgrid.Key)
		var key any
		if e == nil {
			key, e = x509.ParsePKIXPublicKey(b)
		}

		_, ok := key.(*ecdsa.PublicKey)
		check(e == nil && ok, "bounce.sendgrid.key", "must be a base64-encoded ECDSA public key")
	}

	text(s.BounceAzure.SharedSecret, "bounce.azure.shared_secret", webhooks && s.BounceAzure.Enabled)
	if s.BounceAzure.SharedSecretHeader != "" {
		check(settingsHTTPHdr(s.BounceAzure.SharedSecretHeader), "bounce.azure.shared_secret_header", "must be an HTTP header name")
	}

	text(s.BouncePostmark.Username, "bounce.postmark.username", webhooks && s.BouncePostmark.Enabled)
	text(s.BouncePostmark.Password, "bounce.postmark.password", webhooks && s.BouncePostmark.Enabled)
	text(s.BounceForwardEmail.Key, "bounce.forwardemail.key", webhooks && s.BounceForwardEmail.Enabled)
	text(s.BounceLettermint.Key, "bounce.lettermint.key", webhooks && s.BounceLettermint.Enabled)

	// Migrated SES settings may have no topics yet. An empty allowlist safely
	// rejects all notifications until topics are configured.
	for i, arn := range s.BounceSES.TopicARNs {
		parts := strings.SplitN(arn, ":", 6)
		check(len(parts) == 6 && parts[0] == "arn" && parts[1] != "" && parts[2] == "sns" && parts[3] != "" && parts[4] != "" && parts[5] != "",
			fmt.Sprintf("bounce.ses.topic_arns[%d]", i), "must be an SNS topic ARN")
	}

	// Enabled bounce mailboxes need connection details + scan interval.
	clear(seen)
	for i, v := range s.BounceBoxes {
		p := fmt.Sprintf("bounce.mailboxes[%d].", i)
		uuid(v.UUID, p+"uuid")

		enabled := s.BounceEnabled && v.Enabled
		if enabled || v.Type != "" {
			enum(v.Type, p+"type", "pop")
		}

		if enabled || v.Host != "" {
			check(settingsHost(v.Host), p+"host", "must be a hostname or IP address without a port")
		}

		if enabled || v.Port != 0 {
			check(v.Port >= 1 && v.Port <= 65535, p+"port", "must be between 1 and 65535")
		}

		if enabled || v.AuthProtocol != "" {
			enum(v.AuthProtocol, p+"auth_protocol", "none", "userpass")
		}

		text(v.Username, p+"username", enabled && v.AuthProtocol != "none")
		text(v.Password, p+"password", false)

		if v.ReturnPath != "" {
			addr(v.ReturnPath, p+"return_path")
		}

		if enabled || v.ScanInterval != "" {
			duration(v.ScanInterval, p+"scan_interval", time.Minute, math.MaxInt64)
		}
	}

	// Scheduled maintenance.
	if s.MaintenanceDB.Vacuum {
		cron(s.MaintenanceDB.VacuumInterval, "maintenance.db.vacuum_cron_interval")
	}

	return err
}

func settingsURL(value string, relative bool) bool {
	if strings.ContainsAny(value, "\r\n\x00\\ ") {
		return false
	}

	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}

	if relative && u.Scheme == "" && u.Host == "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {

		return true
	}

	if (u.Scheme != "http" && u.Scheme != "https") || !settingsHost(u.Hostname()) {
		return false
	}

	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		return err == nil && port > 0 && port <= 65535
	}

	return !strings.HasSuffix(u.Host, ":")
}

// settingsHost validates a hostname or IP address.
func settingsHost(value string) bool {
	return net.ParseIP(value) != nil || settingsDomain(value)
}

// settingsDomain validates an FQDN domain.
func settingsDomain(value string) bool {
	if value == "" || len(value) > 253 {
		return false
	}

	for label := range strings.SplitSeq(strings.TrimSuffix(value, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}

	return true
}

// settingsDomainPattern validates a domain or a domain pattern with a leading wildcard.
func settingsDomainPattern(value string) bool {
	value, err := idna.Lookup.ToASCII(strings.TrimPrefix(value, "*."))
	return err == nil && settingsDomain(value)
}

// settingsFileExt validates a file extension.
func settingsFileExt(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}

	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}

	return true
}

// settingsHTTPHdr validates an HTTP header name.
func settingsHTTPHdr(value string) bool {
	if value == "" {
		return false
	}

	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}

	return true
}

// settingsDuration validates a duration string and returns its time.Duration.
func settingsDuration(value string) (time.Duration, error) {
	ko := koanf.New(".")
	if err := ko.Set("value", value); err != nil {
		return 0, err
	}

	var out struct{ Value time.Duration }
	err := ko.Unmarshal("", &out)

	return out.Value, err
}

// settingsVals returns a map of JSON raw messages for each setting in the given Settings struct.
func settingsVals(s models.Settings) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var values map[string]json.RawMessage
	err = json.Unmarshal(b, &values)

	return values, err
}

// settingsWithKey builds the settings that would result from changing just one key.
// It replaces that key in a JSON map of the current settings, then decodes the map
// back into Settings{} so the caller can validate the whole configuration set. For example,
// switching upload.provider to "s3" must be rejected if the existing S3 bucket is empty.
// This is used only in the "update settings by key" flow.
func settingsWithKey(cur models.Settings, key string, value json.RawMessage) (models.Settings, error) {
	values, err := settingsVals(cur)
	if err != nil {
		return models.Settings{}, err
	}

	_, ok := values[key]
	if !ok {
		return models.Settings{}, fmt.Errorf("%s: unknown or read-only setting", key)
	}

	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		switch key {
		// Array values.
		case "app.from_email", "app.notify_emails", "privacy.exportable", "privacy.domain_blocklist",
			"privacy.domain_allowlist", "security.trusted_urls", "upload.extensions", "smtp", "messengers", "bounce.mailboxes":
			value = json.RawMessage("[]")
		default:
			return models.Settings{}, fmt.Errorf("%s: must not be null", key)
		}
	}

	values[key] = value
	b, err := json.Marshal(values)
	if err != nil {
		return models.Settings{}, err
	}

	var out models.Settings
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("%s: invalid value type", key)
	}

	return out, nil
}
