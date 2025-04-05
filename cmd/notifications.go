package main

import (
	"regexp"
	"strings"
)

const (
	notifTplImport       = "import-status"
	notifTplCampaign     = "campaign-status"
	notifSubscriberOptin = "subscriber-optin"
	notifSubscriberData  = "subscriber-data"
)

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)
)

// getTplSubject extrcts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func getTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
