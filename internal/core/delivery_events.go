package core

import (
	"fmt"

	"github.com/knadh/listmonk/models"
)

type deliveryEventInsertResult struct {
	CampaignFound  bool   `db:"campaign_found"`
	EventID        int64  `db:"event_id"`
	SubscriberID   int64  `db:"subscriber_id"`
	SubscriberUUID string `db:"subscriber_uuid"`
}

// RecordCampaignDeliveryEvents stores a signed provider batch atomically.
// New bounce events invoke the existing bounce workflow in the same transaction,
// while duplicate provider event IDs do not repeat bounce side effects.
func (c *Core) RecordCampaignDeliveryEvents(events []models.CampaignDeliveryEvent) (models.CampaignDeliveryEventResult, error) {
	var result models.CampaignDeliveryEventResult
	if len(events) == 0 {
		return result, nil
	}

	tx, err := c.db.Beginx()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	eventStmt := tx.Stmtx(c.q.RecordCampaignDeliveryEvent)
	bounceStmt := tx.Stmtx(c.q.RecordBounce)

	for _, event := range events {
		if event.ProviderEventID == "" {
			result.Unattributed++
			continue
		}

		var inserted deliveryEventInsertResult
		if err := eventStmt.Get(&inserted,
			event.CampaignUUID,
			event.SubscriberUUID,
			event.Email,
			event.Provider,
			event.ProviderEventID,
			event.ProviderMessageID,
			event.EventType,
			event.Meta,
			event.OccurredAt,
		); err != nil {
			return result, err
		}

		if !inserted.CampaignFound {
			result.Unattributed++
		}
		if inserted.EventID == 0 {
			result.Duplicates++
			continue
		}
		result.Inserted++

		// An event without a matching campaign cannot contribute to campaign
		// delivery metrics. A newly inserted bounce can still protect the
		// subscriber through the existing email fallback workflow below.
		if event.EventType != models.DeliveryEventBounce || inserted.SubscriberID == 0 {
			continue
		}

		action, ok := c.consts.BounceActions[event.BounceType]
		if !ok {
			return result, fmt.Errorf("invalid bounce type %q", event.BounceType)
		}
		if _, err := bounceStmt.Exec(
			inserted.SubscriberUUID,
			event.Email,
			event.CampaignUUID,
			event.BounceType,
			event.Provider,
			event.Meta,
			event.OccurredAt,
			action.Count,
			action.Action,
		); err != nil {
			return result, err
		}
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
