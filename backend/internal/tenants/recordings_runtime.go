package tenants

func normalizeTenantForRuntime(item tenantResponse, recordingsEnabled bool) tenantResponse {
	if recordingsEnabled {
		return item
	}
	item.RecordingEnabled = false
	item.RecordingRetentionDays = nil
	return item
}
