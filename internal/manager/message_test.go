package manager

import (
	"testing"

	"github.com/knadh/listmonk/models"
)

func TestLettermintCampaignMetadataHeader(t *testing.T) {
	campaign := models.Campaign{
		UUID:        "57b8f91b-9859-41ed-a1db-58ac981f4a45",
		Body:        "Campaign body",
		ContentType: models.CampaignContentTypePlain,
		Headers:     models.Headers{{"X-LM-Metadata-X-Listmonk-Campaign": "{{ .Campaign.UUID }}"}},
	}
	if err := campaign.CompileTemplate(nil); err != nil {
		t.Fatal(err)
	}
	message := CampaignMessage{Campaign: &campaign}
	if err := message.render(); err != nil {
		t.Fatal(err)
	}
	if got := message.headers[0]["X-LM-Metadata-X-Listmonk-Campaign"]; got != campaign.UUID {
		t.Fatalf("campaign metadata header = %q, want %q", got, campaign.UUID)
	}
}
