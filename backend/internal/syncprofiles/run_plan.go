package syncprofiles

import (
	"context"
	"fmt"
	"strings"
)

func (s Service) buildPlan(ctx context.Context, profileID string, devices []discoveredDevice, conflictStrategy string) (syncPlan, error) {
	plan := syncPlan{
		ToCreate: []discoveredDevice{},
		ToUpdate: []syncPlanUpdateItem{},
		ToSkip:   []syncPlanSkipItem{},
		Errors:   []syncPlanErrorItem{},
	}

	rows, err := s.DB.Query(ctx, `
SELECT id, "externalId", name, host, port, type::text
FROM "Connection"
WHERE "syncProfileId" = $1
`, profileID)
	if err != nil {
		return syncPlan{}, fmt.Errorf("list existing sync connections: %w", err)
	}
	defer rows.Close()

	type existingConnection struct {
		ID         string
		ExternalID string
		Name       string
		Host       string
		Port       int
		Type       string
	}
	existing := make(map[string]existingConnection)
	for rows.Next() {
		var item existingConnection
		if err := rows.Scan(&item.ID, &item.ExternalID, &item.Name, &item.Host, &item.Port, &item.Type); err != nil {
			return syncPlan{}, fmt.Errorf("scan existing sync connection: %w", err)
		}
		if strings.TrimSpace(item.ExternalID) != "" {
			existing[item.ExternalID] = item
		}
	}
	if err := rows.Err(); err != nil {
		return syncPlan{}, fmt.Errorf("iterate existing sync connections: %w", err)
	}

	for _, device := range devices {
		if strings.TrimSpace(device.Host) == "" {
			plan.Errors = append(plan.Errors, syncPlanErrorItem{Device: device, Error: "No IP address resolved"})
			continue
		}

		current, ok := existing[device.ExternalID]
		if !ok {
			plan.ToCreate = append(plan.ToCreate, device)
			continue
		}

		if conflictStrategy == "skip" {
			plan.ToSkip = append(plan.ToSkip, syncPlanSkipItem{
				Device: device,
				Reason: "Connection already exists (skip strategy)",
			})
			continue
		}

		changes := make([]string, 0)
		if current.Name != device.Name {
			changes = append(changes, fmt.Sprintf(`name: "%s" → "%s"`, current.Name, device.Name))
		}
		if current.Host != device.Host {
			changes = append(changes, fmt.Sprintf(`host: "%s" → "%s"`, current.Host, device.Host))
		}
		if current.Port != device.Port {
			changes = append(changes, fmt.Sprintf("port: %d → %d", current.Port, device.Port))
		}
		if current.Type != device.Protocol {
			changes = append(changes, fmt.Sprintf("protocol: %s → %s", current.Type, device.Protocol))
		}

		if len(changes) == 0 {
			plan.ToSkip = append(plan.ToSkip, syncPlanSkipItem{
				Device: device,
				Reason: "No changes detected",
			})
			continue
		}
		if conflictStrategy == "update" || conflictStrategy == "overwrite" {
			plan.ToUpdate = append(plan.ToUpdate, syncPlanUpdateItem{
				Device:       device,
				ConnectionID: current.ID,
				Changes:      changes,
			})
		}
	}
	return plan, nil
}
