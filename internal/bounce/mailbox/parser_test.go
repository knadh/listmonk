package mailbox

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/knadh/listmonk/models"
)

func runDSNParserTest(t *testing.T, path string, expected models.Bounce, regexOk bool) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	bounce, err := parseDSN(f)
	if err != nil {
		if !regexOk && err != errorNotMultipartReport {
			t.Errorf("%v", err)
		}
	}
	if bounce.Email == "" && bounce.SubscriberUUID == "" {
		t.Errorf("unable to detrmine subscriber email or UUID")
	}
	if diff := cmp.Diff(expected, bounce); diff != "" {
		t.Errorf("parsing: %s mismatch (-want +got):\n%s", path, diff)
	}
}

func TestParseDSN1(t *testing.T) {
	meta := bounceMeta{
		From:           "Test <hello@example.com>",
		Subject:        "Confirm your email address",
		MessageID:      "<07WQFpqCuYqPnfRIyWPW3E4X@example.com>",
		DeliveredTo:    "bounce@icloud.com",
		Received:       []string{"from substrate.office.com (2603:10a6:600:37c::19) by LOVP265MB8674.GBRP265.PROD.OUTLOOK.COM with HTTP via LO4P265CA0257.GBRP265.PROD.OUTLOOK.COM; Wed, 23 Sep 2026 09:15:43 +0000"},
		ClassifyReason: "smtp_status=5.7.1",
		BounceDetails: []string{
			"smtp;554 5.7.1 [CS01] Message rejected due to local policy. Please visit https://support.apple.com/en-us/HT204137. Txn ID 00000000-0000-0000-0000-000000000000",
			"Remote-MTA: p00-iscream-smtp-7b95d6f6c9-64bmv",
		},
	}
	created, _ := time.Parse(time.RFC3339, "2026-09-23T09:15:43Z")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "a4eec08a-4c6e-4c49-ba0b-54bb2a96d768",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn1.eml", expected, false)
}

func TestParseDSN2(t *testing.T) {
	meta := bounceMeta{
		From:        "<hirlevel@example.com>",
		Subject:     "=?UTF-8?q?Bojler_Elad=C3=B3!?=",
		MessageID:   "<1641477776713090833.30240.5322564447204887968@edu3>",
		DeliveredTo: "test@example.com",
		Received: []string{
			"from localhost (localhost [127.0.0.1]) by mail.example.com (Postfix) with UTF8SMTP id 41FE61A09C01 for <test@example.com>; Thu,  6 Jan 2022 15:02:57 +0100 (CET)",
			"from mail.example.com ([127.0.0.1]) by localhost (mail.example.com [127.0.0.1]) (amavisd-new, port 10026) with UTF8LMTP id P55Eyi_KKrPe for <test@example.com>; Thu,  6 Jan 2022 15:02:57 +0100 (CET)",
			"from localhost (mail.example.com [127.0.0.1]) (Authenticated sender: hirlevel@example.com) by mail.example.com (Postfix) with UTF8SMTPSA id AE0231A086CF for <test@example.com>; Thu,  6 Jan 2022 15:02:56 +0100 (CET)",
		},
		ClassifyReason: "smtp_status=5.4.4",
		BounceDetails:  []string{"X-Postfix; Host or domain name not found. Name service error for name=example.com type=A: Host not found"},
	}
	created, _ := time.Parse(time.RFC3339, "2022-01-06T15:02:57+01:00")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "a1c7a976-ad38-44f2-84de-55b0dc375352",
		CampaignUUID:   "d9510e30-b4e7-4548-b75d-810cb2d03578",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn2.eml", expected, false)
}

func TestParseDSN3(t *testing.T) {
	meta := bounceMeta{
		From:        "<hirlevel@example.com>",
		Subject:     "=?UTF-8?q?Bojler_Elad=C3=B3!?=",
		MessageID:   "<1641477776588637040.30240.3828675836904473490@edu3>",
		DeliveredTo: "test@example.com",
		Received: []string{
			"from localhost (localhost [127.0.0.1]) by mail.example.com (Postfix) with UTF8SMTP id 00E2B1A094C3 for <test@example.com>; Thu,  6 Jan 2022 15:02:57 +0100 (CET)",
			"from mail.example.com ([127.0.0.1]) by localhost (mail.example.com [127.0.0.1]) (amavisd-new, port 10026) with UTF8LMTP id mbyVwqnv5y7G for <test@example.com>; Thu,  6 Jan 2022 15:02:56 +0100 (CET)",
			"from localhost (mail.example.com [127.0.0.1]) (Authenticated sender: hirlevel@example.com) by mail.example.com (Postfix) with UTF8SMTPSA id 8FA301A036EA for <test@example.com>; Thu,  6 Jan 2022 15:02:56 +0100 (CET)",
		},
		ClassifyReason: "smtp_status=5.4.4",
		BounceDetails:  []string{"X-Postfix; Host or domain name not found. Name service error for name=example.com type=A: Host not found"},
	}
	created, _ := time.Parse(time.RFC3339, "2022-01-06T15:02:57+01:00")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "f43f40bf-bcbd-4bd0-b793-2aa9b9a6961d",
		CampaignUUID:   "d9510e30-b4e7-4548-b75d-810cb2d03578",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn3.eml", expected, false)
}

func TestParseDSN4(t *testing.T) {
	meta := bounceMeta{
		From:        "\"Test Test\" <no-reply@sender.example.com>",
		MessageID:   "<1632903633748563659.927.5595426720521411472@examplece>",
		DeliveredTo: "sas@example.ch",
		Received: []string{
			"from mail.example.com (mail.example.com [10.1.2.101]) by mail.example.com with ESMTP id 18T8KXXY004617 for <sas@example.ch>; Wed, 29 Sep 2021 08:20:33 GMT",
		},
		ClassifyReason: "smtp_status=5.1.2",
		BounceDetails: []string{
			"SMTP; 550 Host unknown",
			"Remote-MTA: example.ch",
		},
	}
	created, _ := time.Parse(time.RFC3339, "2021-09-29T09:20:33+01:00")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "73d11b6b-3da2-47d5-9dd6-7b3c96a8e15b",
		CampaignUUID:   "730438ad-a86a-424d-8604-288f6f6f1831",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn4.eml", expected, false)
}

func TestParseDSN5(t *testing.T) {
	meta := bounceMeta{
		From:        "\"Test\" <hello@example.com>",
		Subject:     "Confirm your email address",
		MessageID:   "<xy4svdnGpJwy2BjYjJouGako@example.com>",
		DeliveredTo: "bounce@icloud.com",
		Received: []string{
			"from list.example.com by list.example.com ([172.30.0.3]) via tcp with ESMTPSA id flM-mvBoUb3HgCXYACQp7w (TLS1.3 TLS_AES_128_GCM_SHA256) for <bounce@icloud.com>; 30 Sep 2026 15:02:58 +0000",
		},
		ClassifyReason: "smtp_status=5.7.1",
		BounceDetails: []string{
			"smtp; 554 5.7.1 [CS01] Message rejected due to local policy. Please visit https://support.apple.com/en-us/HT204137. Txn ID 00000000-0000-0000-0000-000000000000",
			"Remote-MTA: mx01.mail.icloud.com",
		},
	}
	created, _ := time.Parse(time.RFC3339, "2026-09-30T15:02:58Z")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "a4eec08a-4c6e-4c49-ba0b-54bb2a96d768",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn5.eml", expected, false)
}

func TestParseDSN6(t *testing.T) {
	meta := bounceMeta{
		MessageID:      "<xy4svdnGpJwy2BjYjJouGako@example.com>",
		ClassifyReason: "smtp_status=5.7.1",
		BounceDetails:  []string{"smtp; 554 5.7.1", "Remote-MTA: mx01.mail.icloud.com"},
	}
	created, _ := time.Parse(time.RFC3339, "2026-09-30T15:02:58Z")
	expected := models.Bounce{
		Type:      models.BounceTypeHard,
		CreatedAt: created,
		Email:     "bounce@icloud.com",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn6.eml", expected, false)
}

func TestParseDSN7(t *testing.T) {
	meta := bounceMeta{
		From:           "Test Bounces <bounces@example.com>",
		Subject:        "Test Message",
		MessageID:      "<010b01a10bd5988a-600e3591-2708-4a3c-8c35-f0825c9ab130-000000@eu-west-2.amazonses.com>",
		ClassifyReason: "smtp_status=5.1.1",
		BounceDetails: []string{
			"smtp; 550 5.1.1 As requested: user unknown <bounce@simulator.amazonses.com>",
		},
	}
	created, _ := time.Parse(time.RFC3339, "2026-10-05T11:31:50Z")
	expected := models.Bounce{
		Type:      models.BounceTypeHard,
		CreatedAt: created,
		Email:     "bounce@simulator.amazonses.com",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn7.eml", expected, false)
}

func TestParseDSN8(t *testing.T) {
	meta := bounceMeta{
		From:      "Test <hello@example.com>",
		Subject:   "Confirm your email address",
		MessageID: "<07WQFpqCuYqPnfRIyWPW3E4X@example.com>",
		Received: []string{
			"from substrate.office.com (2603:10a6:600:37c::19) by LOVP265MB8674.GBRP265.PROD.OUTLOOK.COM with HTTP via LO4P265CA0257.GBRP265.PROD.OUTLOOK.COM; Wed, 23 Sep 2026 09:15:43 +0000",
			"from substrate.office.com (2603:10a6:600:37c::19) by LOVP265MB8674.GBRP265.PROD.OUTLOOK.COM with HTTP via LO4P265CA0257.GBRP265.PROD.OUTLOOK.COM; Wed, 23 Sep 2026 09:15:43 +0000",
		},
		ClassifyReason: "smtp_status=5.7.1",
	}
	created, _ := time.Parse(time.RFC3339, "2026-09-23T09:15:51Z")
	expected := models.Bounce{
		Type:           models.BounceTypeHard,
		CreatedAt:      created,
		SubscriberUUID: "a4eec08a-4c6e-4c49-ba0b-54bb2a96d768",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/dsn8.eml", expected, true)
}

func TestParseARF1(t *testing.T) {
	meta := bounceMeta{
		From:      "Bounce Test <bounces@example.com>",
		Subject:   "Test Message",
		MessageID: "<010b01a10bd62800-41b17c07-f21d-44fe-8e92-7d52e2484de7-000000@eu-west-2.amazonses.com>",
		Received: []string{
			"from d218-16.smtp-out.eu-west-2.amazonses.com (ip-10-0-80-40.eu-west-1.compute.internal [10.0.80.40]) by ip-10-0-61-199.eu-west-1.compute.internal with SMTP (Amazon SES Mailbox Simulator) id 6YUvyUTg3xyPX18uJ6mX for complaint@simulator.amazonses.com; Mon, 05 Oct 2026 11:32:27 +0000 (UTC)",
		},
		ClassifyReason: "rfc5965 abuse report",
		BounceDetails: []string{
			"User-Agent: Amazon SES Mailbox Simulator",
		},
	}
	created, _ := time.Parse(time.RFC3339, "2026-10-05T11:32:26Z")
	expected := models.Bounce{
		Type:      models.BounceTypeHard,
		CreatedAt: created,
		Email:     "complaint@simulator.amazonses.com",
	}
	expected.Meta, _ = json.Marshal(meta)
	runDSNParserTest(t, "test-messages/arf1.eml", expected, false)
}
