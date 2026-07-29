package email

import (
	"encoding/json"
	"net/textproto"
	"reflect"
	"testing"
)

func TestIsSendGridHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "smtp.sendgrid.net", want: true},
		{host: " SMTP.SENDGRID.NET ", want: true},
		{host: "smtp.smtp.com", want: false},
		{host: "", want: false},
	}

	for _, tt := range tests {
		if got := isSendGridHost(tt.host); got != tt.want {
			t.Errorf("isSendGridHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestSetSendGridCampaignHeader(t *testing.T) {
	const campaignUUID = "d6da0074-1084-4aa1-9c62-65fde375d33c"

	tests := []struct {
		name     string
		existing string
		want     map[string]any
	}{
		{
			name: "adds campaign metadata",
			want: map[string]any{
				"unique_args": map[string]any{
					"XListmonkCampaign": campaignUUID,
				},
			},
		},
		{
			name:     "preserves existing SMTPAPI fields",
			existing: `{"category":["newsletter"],"unique_args":{"tenant":"kvsocial"}}`,
			want: map[string]any{
				"category": []any{"newsletter"},
				"unique_args": map[string]any{
					"tenant":            "kvsocial",
					"XListmonkCampaign": campaignUUID,
				},
			},
		},
		{
			name:     "overrides stale campaign metadata",
			existing: `{"unique_args":{"XListmonkCampaign":"old-value"}}`,
			want: map[string]any{
				"unique_args": map[string]any{
					"XListmonkCampaign": campaignUUID,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := textproto.MIMEHeader{}
			if tt.existing != "" {
				headers.Set(hdrSendGridSMTPAPI, tt.existing)
			}

			if err := setSendGridCampaignHeader(headers, campaignUUID); err != nil {
				t.Fatalf("setSendGridCampaignHeader() error = %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(headers.Get(hdrSendGridSMTPAPI)), &got); err != nil {
				t.Fatalf("generated header is not valid JSON: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generated payload = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSetSendGridCampaignHeaderRejectsInvalidPayloads(t *testing.T) {
	tests := []string{
		`not-json`,
		`{"unique_args":"not-an-object"}`,
	}

	for _, existing := range tests {
		headers := textproto.MIMEHeader{}
		headers.Set(hdrSendGridSMTPAPI, existing)

		if err := setSendGridCampaignHeader(headers, "campaign-uuid"); err == nil {
			t.Errorf("setSendGridCampaignHeader(%q) expected an error", existing)
		}
	}
}

func TestSetSendGridCampaignHeaderIgnoresEmptyCampaignUUID(t *testing.T) {
	headers := textproto.MIMEHeader{}
	if err := setSendGridCampaignHeader(headers, ""); err != nil {
		t.Fatalf("setSendGridCampaignHeader() error = %v", err)
	}
	if got := headers.Get(hdrSendGridSMTPAPI); got != "" {
		t.Errorf("unexpected header for empty campaign UUID: %q", got)
	}
}
