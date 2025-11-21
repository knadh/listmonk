package models

// Enum values for various statuses.
const (
	// Headers attached to e-mails for bounce tracking.
	EmailHeaderSubscriberUUID = "X-Listmonk-Subscriber"
	EmailHeaderCampaignUUID   = "X-Listmonk-Campaign"

	// Standard e-mail headers.
	EmailHeaderDate        = "Date"
	EmailHeaderFrom        = "From"
	EmailHeaderSubject     = "Subject"
	EmailHeaderMessageId   = "Message-Id"
	EmailHeaderDeliveredTo = "Delivered-To"
	EmailHeaderReceived    = "Received"

	// TwoFA types.
	TwofaTypeNone = "none"
	TwofaTypeTOTP = "totp"
)
