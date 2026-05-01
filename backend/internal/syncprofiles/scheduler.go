package syncprofiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	cron "github.com/robfig/cron/v3"
)

type schedulerState struct {
	mu      sync.Mutex
	parser  cron.Parser
	cron    *cron.Cron
	entries map[string]cron.EntryID
}

func NewSchedulerState() *schedulerState {
	return &schedulerState{
		parser: cron.NewParser(
			cron.SecondOptional |
				cron.Minute |
				cron.Hour |
				cron.Dom |
				cron.Month |
				cron.Dow |
				cron.Descriptor,
		),
		entries: make(map[string]cron.EntryID),
	}
}

func (s Service) StartScheduler(ctx context.Context) error {
	if s.Scheduler == nil || s.DB == nil {
		return nil
	}
	s.ensureSchedulerStarted()

	rows, err := s.DB.Query(ctx, `
SELECT id, "cronExpression", enabled
FROM "SyncProfile"
WHERE "cronExpression" IS NOT NULL
`)
	if err != nil {
		return fmt.Errorf("list scheduled sync profiles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			profileID      string
			cronExpression sql.NullString
			enabled        bool
		)
		if err := rows.Scan(&profileID, &cronExpression, &enabled); err != nil {
			return fmt.Errorf("scan scheduled sync profile: %w", err)
		}
		if cronExpression.Valid {
			expr := cronExpression.String
			s.reconcileSchedule(profileID, &expr, enabled)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate scheduled sync profiles: %w", err)
	}
	return nil
}

func (s Service) StopScheduler() {
	if s.Scheduler == nil {
		return
	}

	s.Scheduler.mu.Lock()
	defer s.Scheduler.mu.Unlock()

	if s.Scheduler.cron != nil {
		ctx := s.Scheduler.cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
	s.Scheduler.cron = nil
	s.Scheduler.entries = make(map[string]cron.EntryID)
}

func (s Service) reconcileSchedule(profileID string, cronExpression *string, enabled bool) {
	if s.Scheduler == nil {
		return
	}

	normalized := normalizeCronExpression(cronExpression)
	if !enabled || normalized == nil {
		s.unregisterSchedule(profileID)
		return
	}
	if !s.isValidCronExpression(*normalized) {
		s.unregisterSchedule(profileID)
		log.Printf("syncprofiles: invalid cron expression %q for profile %s", *normalized, profileID)
		return
	}
	logScheduleError(s.registerSchedule(profileID, *normalized))
}

func (s Service) isValidCronExpression(expr string) bool {
	if s.Scheduler == nil {
		return false
	}
	_, err := s.Scheduler.parser.Parse(strings.TrimSpace(expr))
	return err == nil
}

func (s Service) ensureSchedulerStarted() {
	if s.Scheduler == nil {
		return
	}

	s.Scheduler.mu.Lock()
	defer s.Scheduler.mu.Unlock()

	if s.Scheduler.cron != nil {
		return
	}
	s.Scheduler.cron = cron.New(
		cron.WithParser(s.Scheduler.parser),
		cron.WithLocation(time.UTC),
	)
	s.Scheduler.cron.Start()
}

func (s Service) registerSchedule(profileID, cronExpression string) error {
	if s.Scheduler == nil {
		return nil
	}
	s.ensureSchedulerStarted()

	s.Scheduler.mu.Lock()
	defer s.Scheduler.mu.Unlock()

	if entryID, ok := s.Scheduler.entries[profileID]; ok && s.Scheduler.cron != nil {
		s.Scheduler.cron.Remove(entryID)
		delete(s.Scheduler.entries, profileID)
	}

	if s.Scheduler.cron == nil {
		return nil
	}
	entryID, err := s.Scheduler.cron.AddFunc(cronExpression, func() {
		if err := s.runScheduledSync(profileID); err != nil {
			log.Printf("syncprofiles: scheduled sync failed for profile %s: %v", profileID, err)
		}
	})
	if err != nil {
		return fmt.Errorf("register sync schedule for %s: %w", profileID, err)
	}
	s.Scheduler.entries[profileID] = entryID
	return nil
}

func (s Service) unregisterSchedule(profileID string) {
	if s.Scheduler == nil {
		return
	}

	s.Scheduler.mu.Lock()
	defer s.Scheduler.mu.Unlock()

	entryID, ok := s.Scheduler.entries[profileID]
	if !ok || s.Scheduler.cron == nil {
		return
	}
	s.Scheduler.cron.Remove(entryID)
	delete(s.Scheduler.entries, profileID)
}

func (s Service) runScheduledSync(profileID string) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}

	var (
		tenantID    string
		createdByID string
		enabled     bool
		name        string
	)
	err := s.DB.QueryRow(context.Background(), `
SELECT "tenantId", "createdById", enabled, name
FROM "SyncProfile"
WHERE id = $1
`, profileID).Scan(&tenantID, &createdByID, &enabled, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.unregisterSchedule(profileID)
			return nil
		}
		return fmt.Errorf("load scheduled sync profile: %w", err)
	}
	if !enabled {
		s.unregisterSchedule(profileID)
		return nil
	}

	log.Printf("syncprofiles: running scheduled sync for profile %q (%s)", name, profileID)
	_, err = s.TriggerSync(context.Background(), createdByID, tenantID, profileID, false)
	return err
}
