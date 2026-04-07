package sshsessions

import (
	"slices"
	"time"
)

func (t *activeDBTunnel) snapshot() dbTunnelListItem {
	t.mu.Lock()
	defer t.mu.Unlock()

	return dbTunnelListItem{
		TunnelID:         t.ID,
		SessionID:        t.SessionID,
		LocalHost:        "127.0.0.1",
		LocalPort:        t.LocalPort,
		TargetDBHost:     t.TargetDBHost,
		TargetDBPort:     t.TargetDBPort,
		DBType:           cloneStringPtr(t.DBType),
		ConnectionString: cloneStringPtr(t.ConnectionString),
		ConnectionID:     t.ConnectionID,
		Healthy:          t.healthy,
		CreatedAt:        t.CreatedAt,
		LastError:        cloneStringPtr(t.lastError),
		LastUsedAt:       cloneTimePtr(t.LastUsedAt),
	}
}

func (t *activeDBTunnel) setForwardError(err error) {
	if err == nil {
		return
	}
	message := err.Error()
	now := time.Now().UTC()

	t.mu.Lock()
	defer t.mu.Unlock()
	t.healthy = false
	t.lastError = &message
	t.LastUsedAt = &now
}

func (t *activeDBTunnel) touch() {
	now := time.Now().UTC()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.LastUsedAt = &now
}

func (t *activeDBTunnel) close() {
	t.closeOnce.Do(func() {
		_ = t.listener.Close()
		_ = t.sshClient.Close()
	})
}

func (r *dbTunnelRegistry) add(tunnel *activeDBTunnel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tunnels[tunnel.ID] = tunnel
}

func (r *dbTunnelRegistry) get(id string) (*activeDBTunnel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tunnel, ok := r.tunnels[id]
	return tunnel, ok
}

func (r *dbTunnelRegistry) listForUser(userID string) []dbTunnelListItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]dbTunnelListItem, 0, len(r.tunnels))
	for _, tunnel := range r.tunnels {
		if tunnel.UserID == userID {
			items = append(items, tunnel.snapshot())
		}
	}
	slices.SortFunc(items, func(a, b dbTunnelListItem) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return items
}

func (r *dbTunnelRegistry) closeOwned(tunnelID, userID string) (*activeDBTunnel, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tunnel, ok := r.tunnels[tunnelID]
	if !ok || tunnel.UserID != userID {
		return nil, false
	}
	delete(r.tunnels, tunnelID)
	return tunnel, true
}

func (r *dbTunnelRegistry) closeByID(tunnelID string) {
	r.mu.Lock()
	tunnel, ok := r.tunnels[tunnelID]
	if ok {
		delete(r.tunnels, tunnelID)
	}
	r.mu.Unlock()
	if ok {
		tunnel.close()
	}
}
