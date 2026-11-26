package main

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/models"
)

func defaultSettings(t *testing.T) models.Settings {
	t.Helper()

	b, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Read the shipped defaults so a schema change also exercises validation.
	b = b[strings.Index(string(b), "INSERT INTO settings (key, value) VALUES"):]
	b = b[:strings.Index(string(b), ";")]
	values := make(map[string]json.RawMessage)
	for _, m := range regexp.MustCompile(`(?s)\('([^']+)',\s*'((?:[^']|'')*)'\)`).FindAllSubmatch(b, -1) {
		values[string(m[1])] = json.RawMessage(strings.ReplaceAll(string(m[2]), "''", "'"))
	}
	b, err = json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	var s models.Settings
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func settingsTestApp(t *testing.T) *App {
	t.Helper()

	i, err := i18n.New([]byte(`{"_.code":"en", "_.name":"English"}`))
	if err != nil {
		t.Fatal(err)
	}
	return &App{i18n: i}
}

func TestSettingsDefaults(t *testing.T) {
	s := defaultSettings(t)
	if err := validateSettings(s); err != nil {
		t.Fatal(err)
	}
	if err := settingsTestApp(t).prepareSettings(&s, s); err != nil {
		t.Fatal(err)
	}
	if err := validateSettings(s); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsValidation(t *testing.T) {
	f := func(change func(*models.Settings), field string) {
		t.Helper()

		s := defaultSettings(t)
		change(&s)
		err := validateSettings(s)
		if field == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.HasPrefix(err.Error(), field+":") {
			t.Fatalf("expected %s error, got %v", field, err)
		}
	}

	f(func(s *models.Settings) { s.AppSiteName = " " }, "app.site_name")
	f(func(s *models.Settings) { s.AppRootURL = "https://example.com/news" }, "")
	f(func(s *models.Settings) { s.AppRootURL = "https://example.com/?x=1" }, "app.root_url")
	f(func(s *models.Settings) { s.AppLogoURL = "/logo.svg" }, "")
	f(func(s *models.Settings) { s.AppNotifyEmails = []string{"bad"} }, "app.notify_emails[0]")
	f(func(s *models.Settings) { s.AppFromEmail = nil }, "app.from_email")
	f(func(s *models.Settings) { s.AppLang = "../en" }, "app.lang")
	f(func(s *models.Settings) { s.AppBatchSize = 0 }, "app.batch_size")
	f(func(s *models.Settings) { s.AppBatchSize = 100001 }, "app.batch_size")
	f(func(s *models.Settings) { s.AppConcurrency = math.MaxInt }, "app.concurrency")
	f(func(s *models.Settings) { s.AppMessageRate = 10001 }, "app.message_rate")
	f(func(s *models.Settings) { s.AppMessageRate = 10000 }, "")
	f(func(s *models.Settings) { s.AppMaxSendErrors = -1 }, "app.max_send_errors")
	f(func(s *models.Settings) { s.AppMaxSendErrors = 0 }, "")
	f(func(s *models.Settings) { s.AppMessageSlidingWindow = true; s.AppMessageSlidingWindowRate = 0 }, "app.message_sliding_window_rate")
	f(func(s *models.Settings) { s.AppMessageSlidingWindow = true; s.AppMessageSlidingWindowDuration = "1s" }, "app.message_sliding_window_duration")
	f(func(s *models.Settings) { s.CacheSlowQueries = true; s.CacheSlowQueriesInterval = "bad" }, "app.cache_slow_queries_interval")
	f(func(s *models.Settings) { s.PrivacyExportable = []string{"password"} }, "privacy.exportable[0]")
	f(func(s *models.Settings) { s.DomainAllowlist = []string{"https://example.com"} }, "privacy.domain_allowlist[0]")
	f(func(s *models.Settings) { s.DomainBlocklist = []string{"example.com"} }, "")
	f(func(s *models.Settings) { s.SecuritySignedLinks.TTL = "0" }, "")
	f(func(s *models.Settings) { s.SecuritySignedLinks.TTL = "nonsense" }, "security.signed_links.ttl")
	f(func(s *models.Settings) { s.SecuritySessionDuration = "0" }, "security.session_duration")
	f(func(s *models.Settings) { s.SecurityTrustedURLs = []string{"*", "https://example.com"} }, "")
	f(func(s *models.Settings) {
		s.SecurityCaptcha.Altcha.Enabled = true
		s.SecurityCaptcha.Altcha.Complexity = 0
	}, "security.captcha.altcha.complexity")
	f(func(s *models.Settings) { s.SecurityCaptcha.HCaptcha.Enabled = true }, "security.captcha.hcaptcha.key")
	f(func(s *models.Settings) { s.OIDC.Enabled = true }, "security.oidc.provider_url")
	f(func(s *models.Settings) { s.UploadProvider = "unknown" }, "upload.provider")
	f(func(s *models.Settings) { s.UploadMaxFileSize = math.MaxInt64 }, "upload.max_file_size")
	f(func(s *models.Settings) { s.UploadExtensions = []string{"../png"} }, "upload.extensions[0]")
	f(func(s *models.Settings) { s.UploadFilesystem.UploadURI = "/" }, "upload.filesystem.upload_uri")
	f(func(s *models.Settings) { s.UploadS3.Bucket = "my-bucket"; s.UploadProvider = "s3" }, "")
	f(func(s *models.Settings) {
		s.UploadS3.Bucket = "my-bucket"
		s.UploadProvider = "s3"
		s.UploadS3.AwsAccessKeyID = "key"
	}, "upload.s3.aws_secret_access_key")
	f(func(s *models.Settings) {
		s.UploadProvider = "s3"
		s.UploadS3.Bucket = "test"
		s.UploadS3.Expiry = "169h"
	}, "upload.s3.expiry")
	f(func(s *models.Settings) { s.SMTP[0].Port = 65536 }, "smtp[0].port")
	f(func(s *models.Settings) { s.SMTP[0].Host = "smtp.example.com:25" }, "smtp[0].host")
	f(func(s *models.Settings) { s.SMTP[0].AuthProtocol = "invalid" }, "smtp[0].auth_protocol")
	f(func(s *models.Settings) { s.SMTP[0].TLSType = "starttls" }, "smtp[0].tls_type")
	f(func(s *models.Settings) { s.SMTP[0].MaxConns = 0 }, "smtp[0].max_conns")
	f(func(s *models.Settings) { s.SMTP[0].MaxMsgRetries = -1 }, "smtp[0].max_msg_retries")
	f(func(s *models.Settings) { s.SMTP[0].WaitTimeout = "oops" }, "smtp[0].wait_timeout")
	f(func(s *models.Settings) {
		s.SMTP[0].EmailHeaders = []map[string]string{{"X-Test": "ok\r\nBcc: other@example.com"}}
	}, "smtp[0].email_headers[0]")
	f(func(s *models.Settings) { s.SMTP[0].FromAddresses = []string{"https://example.com"} }, "smtp[0].from_addresses[0]")
	f(func(s *models.Settings) { s.SMTP[0].UUID = "bad" }, "smtp[0].uuid")
	f(func(s *models.Settings) {
		s.SMTP[0].UUID = "acbd1234-1111-4111-8111-111111111111"
		s.SMTP[1].UUID = strings.ToUpper(s.SMTP[0].UUID)
	}, "smtp[1].uuid")
	f(func(s *models.Settings) {
		s.SMTP[1].Host = ""
		s.SMTP[1].Port = 0
		s.SMTP[1].WaitTimeout = ""
		s.SMTP[1].IdleTimeout = ""
		s.SMTP[1].MaxConns = 0
	}, "")
	f(func(s *models.Settings) { v := s.BounceActions["hard"]; v.Count = 0; s.BounceActions["hard"] = v }, "bounce.actions.hard.count")
	f(func(s *models.Settings) { delete(s.BounceActions, "soft") }, "bounce.actions.soft")
	f(func(s *models.Settings) {
		s.BounceEnabled = true
		s.BounceEnableWebhooks = true
		s.BounceSendgrid.Enabled = true
	}, "bounce.sendgrid.key")
	f(func(s *models.Settings) { s.BounceSES.TopicARNs = []string{"bad"} }, "bounce.ses.topic_arns[0]")
	f(func(s *models.Settings) { s.BounceBoxes[0].ScanInterval = "59s" }, "bounce.mailboxes[0].scan_interval")
	f(func(s *models.Settings) { s.MaintenanceDB.Vacuum = true; s.MaintenanceDB.VacuumInterval = "bad" }, "maintenance.db.vacuum_cron_interval")
	f(func(s *models.Settings) { s.AdminCustomJS = "console.log('hello');\n" }, "")
}

func TestSettingsURLs(t *testing.T) {
	f := func(value string, relative, want bool) {
		t.Helper()
		if got := settingsURL(value, relative); got != want {
			t.Fatalf("settingsURL(%q, %v) = %v, want %v", value, relative, got, want)
		}
	}

	f("http://localhost:9000/news", false, true)
	f("https://[::1]:443/news", false, true)
	f("/uploads/logo.svg", true, true)
	f("/uploads/logo.svg", false, false)
	f("//evil.example.com", true, false)
	f("javascript:alert(1)", true, false)
	f("https://user:password@example.com", false, false)
	f("https://example.com:0", false, false)
	f("https://example.com:65536", false, false)
	f("https://example.com:", false, false)
	f("https://example.com/\r\n", false, false)
}

func TestSettingsDurations(t *testing.T) {
	f := func(value string, want time.Duration, valid bool) {
		t.Helper()
		got, err := settingsDuration(value)
		if (err == nil) != valid || valid && got != want {
			t.Fatalf("settingsDuration(%q) = %v, %v", value, got, err)
		}
	}

	f("7d", 7*24*time.Hour, true)
	f("120d", 120*24*time.Hour, true)
	f("1.5d2h", 38*time.Hour, true)
	f("0", 0, true)
	f("10ms", 10*time.Millisecond, true)
	f("", 0, false)
	f("garbage", 0, false)
	f("999999999999d", 0, false)
}

func TestSettingsPreparation(t *testing.T) {
	a := settingsTestApp(t)
	cur := defaultSettings(t)
	cur.SMTP[0].UUID = "acbd1234-1111-4111-8111-111111111111"
	cur.SMTP[0].Password = "secret"
	cur.OIDC.ClientSecret = "oidc-secret"
	cur.OIDC.Roles = []models.OIDCRoleMapping{{Claim: "groups", Match: "staff"}}

	f := func(change func(*models.Settings), wantErr bool) models.Settings {
		t.Helper()
		s := defaultSettings(t)
		change(&s)
		if err := a.prepareSettings(&s, cur); (err != nil) != wantErr {
			t.Fatalf("prepareSettings: %v", err)
		}
		return s
	}

	s := f(func(s *models.Settings) {
		s.SMTP[0].UUID = cur.SMTP[0].UUID
		s.SMTP[0].Password = ""
		s.SMTP[0].Name = " Main "
		s.SMTP[0].Host = " smtp.example.com "
		s.OIDC.Roles = nil
		s.UploadExtensions = []string{" .PNG "}
	}, false)
	if s.SMTP[0].Password != "secret" || s.OIDC.ClientSecret != "oidc-secret" || !reflect.DeepEqual(s.OIDC.Roles, cur.OIDC.Roles) {
		t.Fatal("omitted secrets or role mappings were lost")
	}
	if s.SMTP[0].Name != "email-main" || s.SMTP[0].Host != "smtp.example.com" || s.UploadExtensions[0] != "png" {
		t.Fatal("settings were not normalized")
	}
	s = f(func(s *models.Settings) { s.OIDC.Roles = []models.OIDCRoleMapping{} }, false)
	if len(s.OIDC.Roles) != 0 {
		t.Fatal("explicit empty mapping was not retained")
	}
	f(func(s *models.Settings) { s.SMTP[0].Name = "a b"; s.SMTP[1].Name = "a-b" }, true)
	f(func(s *models.Settings) { s.SMTP[0].Enabled = false }, true)
}

func TestSettingsSingleKey(t *testing.T) {
	cur := defaultSettings(t)
	f := func(key, value string, wantErr bool) models.Settings {
		t.Helper()
		s, err := settingsWithKey(cur, key, json.RawMessage(value))
		if err == nil {
			err = settingsTestApp(t).prepareSettings(&s, s)
		}
		if err == nil {
			err = validateSettings(s)
		}
		if (err != nil) != wantErr {
			t.Fatalf("%s = %s: %v", key, value, err)
		}
		return s
	}

	f("unknown", `1`, true)
	f("internal.secret_key", `"secret"`, true)
	f("app.message_rate", `"10"`, true)
	f("app.message_rate", `null`, true)
	f("app.message_rate", `0`, true)
	f("app.message_rate", `10001`, true)
	f("upload.provider", `"s3"`, true)
	f("smtp", `null`, true)
	s := f("app.site_name", `"Updated"`, false)
	if s.AppSiteName != "Updated" || s.AppMessageRate != cur.AppMessageRate {
		t.Fatal("single-key overlay changed unrelated settings")
	}
	s = f("app.notify_emails", `null`, false)
	if s.AppNotifyEmails == nil || len(s.AppNotifyEmails) != 0 {
		t.Fatal("null was not converted to an empty array")
	}
	cur.OIDC.ClientSecret = "secret"
	s = f("security.oidc", `{"enabled":false}`, false)
	if s.OIDC.ClientSecret != "" {
		t.Fatal("single-key replacement unexpectedly preserved a secret")
	}
	s = f("upload.extensions", `[" .PNG "]`, false)
	values, err := settingsVals(s)
	if err != nil || string(values["upload.extensions"]) != `["png"]` {
		t.Fatalf("single-key write did not use normalized value: %s, %v", values["upload.extensions"], err)
	}
}

func TestSettingsIntegrations(t *testing.T) {
	f := func(key, value, field string) {
		t.Helper()
		s, err := settingsWithKey(defaultSettings(t), key, json.RawMessage(value))
		if err != nil {
			t.Fatal(err)
		}
		err = validateSettings(s)
		if field == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.HasPrefix(err.Error(), field+":") {
			t.Fatalf("expected %s error, got %v", field, err)
		}
	}

	f("messengers", `[{"name":"sms", "enabled":true, "root_url":"https://example.com", "max_conns":10, "timeout":"5s"}]`, "")
	f("messengers", `[{"name":"sms", "enabled":false}]`, "")
	f("messengers", `[{"name":"sms", "enabled":true}]`, "messengers[0].root_url")
	f("messengers", `[{"name":"sms", "timeout":"bad"}]`, "messengers[0].timeout")
	f("messengers", `[{"name":"sms", "max_conns":-1}]`, "messengers[0].max_conns")
	f("messengers", `[{"name":"sms", "max_msg_retries":-1}]`, "messengers[0].max_msg_retries")
	f("security.oidc", `{"enabled":true,"provider_url":"https://example.com","redirect_url":"https://listmonk.example.com/auth/oidc","client_id":"client","client_secret":"secret"}`, "")
	f("security.oidc", `{"enabled":false,"provider_url":"not-a-url"}`, "security.oidc.provider_url")
	f("security.oidc", `{"roles":[{"claim":" ","match":"staff"}]}`, "security.oidc.roles[0].claim")
	f("bounce.sendgrid", `{"key":"bm90LWEta2V5"}`, "bounce.sendgrid.key")
	f("bounce.azure", `{"enabled":true,"shared_secret":"secret"}`, "")
	f("bounce.azure", `{"shared_secret_header":"X Bad Header"}`, "bounce.azure.shared_secret_header")
	f("bounce.ses", `{"topic_arns":["arn:aws:sns:ap-south-1:123456789012:bounces"]}`, "")
	f("smtp", `[{"enabled":false,"from_addresses":["Display <user@example.com>"]}]`, "smtp[0].from_addresses[0]")
}

func TestSettingsCompatibility(t *testing.T) {
	for _, rule := range []string{"example.com", "*.example.com", "例え.jp", "*.例え.jp"} {
		for _, key := range []string{"privacy.domain_allowlist", "privacy.domain_blocklist"} {
			t.Run(key+"/"+rule, func(t *testing.T) {
				b, _ := json.Marshal([]string{rule})
				s, err := settingsWithKey(defaultSettings(t), key, b)
				if err != nil {
					t.Fatal(err)
				}
				if err := validateSettings(s); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	for _, rule := range []string{"*", "*.*.example.com", "foo*.example.com", "https://example.com", "foo bar.com", "-example.com"} {
		if settingsDomainPattern(rule) {
			t.Errorf("accepted invalid domain rule %q", rule)
		}
	}
	for _, kind := range []string{"soft", "hard", "complaint"} {
		s := defaultSettings(t)
		v := s.BounceActions[kind]
		v.Action = "unsubscribe"
		s.BounceActions[kind] = v
		if err := validateSettings(s); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{"0", "0s", "500ms"} {
		s := defaultSettings(t)
		s.SMTP[0].WaitTimeout = value
		s.UploadProvider = "s3"
		s.UploadS3.Bucket = "test"
		s.UploadS3.Expiry = value
		if err := validateSettings(s); err != nil {
			t.Fatal(err)
		}
	}
	for _, expiry := range []string{"", "-1s", "169h", "invalid"} {
		s := defaultSettings(t)
		s.UploadProvider = "s3"
		s.UploadS3.Bucket = "test"
		s.UploadS3.Expiry = expiry
		if err := validateSettings(s); err == nil {
			t.Errorf("accepted invalid active S3 expiry %q", expiry)
		}
	}
	for _, enabled := range []bool{false, true} {
		for _, key := range []string{"app.cache_slow_queries_interval", "app.message_sliding_window_duration", "maintenance.db", "upload.s3"} {
			s := defaultSettings(t)
			s.CacheSlowQueries = enabled
			s.AppMessageSlidingWindow = enabled
			s.MaintenanceDB.Vacuum = enabled
			if enabled {
				s.UploadProvider = "s3"
				s.UploadS3.Bucket = "test"
			}
			switch key {
			case "app.cache_slow_queries_interval":
				s.CacheSlowQueriesInterval = "invalid"
			case "app.message_sliding_window_duration":
				s.AppMessageSlidingWindowDuration = "invalid"
			case "maintenance.db":
				s.MaintenanceDB.VacuumInterval = "invalid"
			case "upload.s3":
				s.UploadS3.Expiry = "720h"
			}
			if err := validateSettings(s); (err != nil) != enabled {
				t.Errorf("%s enabled=%v: %v", key, enabled, err)
			}
		}
	}
	s := defaultSettings(t)
	s.BounceEnabled = true
	s.BounceEnableWebhooks = true
	s.BounceSES.Enabled = true
	if err := validateSettings(s); err != nil {
		t.Fatal(err)
	}
	s.BounceSES.TopicARNs = []string{"invalid"}
	if err := validateSettings(s); err == nil {
		t.Fatal("accepted invalid SNS topic")
	}
	for _, value := range []string{
		`[{"name":"sms","enabled":true,"root_url":"http://localhost/send","timeout":"0","max_conns":0}]`,
		`[{"name":"sms","enabled":true,"root_url":"http://localhost/send","timeout":"-1s","max_conns":0}]`,
		`[{"name":"sms","enabled":true,"root_url":"http://localhost/send","timeout":"0","max_conns":-1}]`,
	} {
		s, err := settingsWithKey(defaultSettings(t), "messengers", json.RawMessage(value))
		if err != nil {
			t.Fatal(err)
		}
		if err := validateSettings(s); (err != nil) != strings.Contains(value, "-1") {
			t.Errorf("%s: %v", value, err)
		}
	}
}
