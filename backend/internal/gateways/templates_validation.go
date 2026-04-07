package gateways

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func normalizeCreateTemplatePayload(input createTemplatePayload) (normalizedCreateTemplatePayload, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return normalizedCreateTemplatePayload{}, &requestError{status: http.StatusBadRequest, message: "name is required"}
	}
	if len(name) > 100 {
		return normalizedCreateTemplatePayload{}, &requestError{status: http.StatusBadRequest, message: "name must be 100 characters or fewer"}
	}

	gatewayType := strings.ToUpper(strings.TrimSpace(input.Type))
	if !isAllowedGatewayType(gatewayType) {
		return normalizedCreateTemplatePayload{}, &requestError{status: http.StatusBadRequest, message: "type must be one of GUACD, SSH_BASTION, MANAGED_SSH, DB_PROXY"}
	}

	host := ""
	if input.Host != nil {
		host = strings.TrimSpace(*input.Host)
	}
	deploymentMode, err := normalizeDeploymentMode(input.DeploymentMode, gatewayType, host)
	if err != nil {
		return normalizedCreateTemplatePayload{}, err
	}

	port := 0
	if input.Port != nil {
		port = *input.Port
	}

	if deploymentModeIsGroup(deploymentMode) && port == 0 {
		switch gatewayType {
		case "MANAGED_SSH":
			port = 2222
		case "GUACD":
			port = 4822
		case "DB_PROXY":
			port = 5432
		}
	}
	if gatewayType == "SSH_BASTION" && port == 0 {
		return normalizedCreateTemplatePayload{}, &requestError{status: http.StatusBadRequest, message: "port is required for SSH_BASTION templates"}
	}

	normalized := normalizedCreateTemplatePayload{
		Name:                     name,
		Type:                     gatewayType,
		Host:                     normalizeGatewayHostForMode(deploymentMode, host),
		Port:                     port,
		DeploymentMode:           deploymentMode,
		Description:              trimStringPtr(input.Description),
		APIPort:                  input.APIPort,
		AutoScale:                input.AutoScale,
		MinReplicas:              input.MinReplicas,
		MaxReplicas:              input.MaxReplicas,
		SessionsPerInstance:      input.SessionsPerInstance,
		ScaleDownCooldownSeconds: input.ScaleDownCooldownSeconds,
		MonitoringEnabled:        input.MonitoringEnabled,
		MonitorIntervalMS:        input.MonitorIntervalMS,
		InactivityTimeoutSeconds: input.InactivityTimeoutSeconds,
		PublishPorts:             input.PublishPorts,
		LBStrategy:               normalizeLBStrategyPtr(input.LBStrategy),
	}
	if err := validateNormalizedCreateTemplatePayload(normalized); err != nil {
		return normalizedCreateTemplatePayload{}, err
	}
	return normalized, nil
}

func validateNormalizedCreateTemplatePayload(input normalizedCreateTemplatePayload) error {
	if input.Port < 1 || input.Port > 65535 {
		return &requestError{status: http.StatusBadRequest, message: "port must be between 1 and 65535"}
	}
	return validateTemplateConstraints(
		input.Description,
		input.APIPort,
		input.MinReplicas,
		input.MaxReplicas,
		input.SessionsPerInstance,
		input.ScaleDownCooldownSeconds,
		input.MonitorIntervalMS,
		input.InactivityTimeoutSeconds,
		input.LBStrategy,
	)
}

func validateUpdateTemplatePayload(input updateTemplatePayload) error {
	if input.Name.Present && input.Name.Value != nil {
		name := strings.TrimSpace(*input.Name.Value)
		if name == "" {
			return &requestError{status: http.StatusBadRequest, message: "name cannot be empty"}
		}
		if len(name) > 100 {
			return &requestError{status: http.StatusBadRequest, message: "name must be 100 characters or fewer"}
		}
	}
	if input.Type.Present && input.Type.Value != nil && !isAllowedGatewayType(strings.ToUpper(strings.TrimSpace(*input.Type.Value))) {
		return &requestError{status: http.StatusBadRequest, message: "type must be one of GUACD, SSH_BASTION, MANAGED_SSH, DB_PROXY"}
	}
	if input.DeploymentMode.Present && input.DeploymentMode.Value != nil {
		switch strings.ToUpper(strings.TrimSpace(*input.DeploymentMode.Value)) {
		case "SINGLE_INSTANCE", "MANAGED_GROUP":
		default:
			return &requestError{status: http.StatusBadRequest, message: "deploymentMode must be SINGLE_INSTANCE or MANAGED_GROUP"}
		}
	}
	return validateTemplateConstraints(
		input.Description.Value,
		input.APIPort.Value,
		input.MinReplicas.Value,
		input.MaxReplicas.Value,
		input.SessionsPerInstance.Value,
		input.ScaleDownCooldownSeconds.Value,
		input.MonitorIntervalMS.Value,
		input.InactivityTimeoutSeconds.Value,
		normalizeLBStrategyPtr(input.LBStrategy.Value),
	)
}

func validateTemplateConstraints(description *string, apiPort, minReplicas, maxReplicas, sessionsPerInstance, scaleDownCooldownSeconds, monitorIntervalMS, inactivityTimeoutSeconds *int, lbStrategy *string) error {
	if description != nil && len(*description) > 500 {
		return &requestError{status: http.StatusBadRequest, message: "description must be 500 characters or fewer"}
	}
	if apiPort != nil && (*apiPort < 1 || *apiPort > 65535) {
		return &requestError{status: http.StatusBadRequest, message: "apiPort must be between 1 and 65535"}
	}
	if minReplicas != nil && (*minReplicas < 0 || *minReplicas > 20) {
		return &requestError{status: http.StatusBadRequest, message: "minReplicas must be between 0 and 20"}
	}
	if maxReplicas != nil && (*maxReplicas < 1 || *maxReplicas > 20) {
		return &requestError{status: http.StatusBadRequest, message: "maxReplicas must be between 1 and 20"}
	}
	if minReplicas != nil && maxReplicas != nil && *minReplicas > *maxReplicas {
		return &requestError{status: http.StatusBadRequest, message: "minReplicas must be less than or equal to maxReplicas"}
	}
	if sessionsPerInstance != nil && (*sessionsPerInstance < 1 || *sessionsPerInstance > 100) {
		return &requestError{status: http.StatusBadRequest, message: "sessionsPerInstance must be between 1 and 100"}
	}
	if scaleDownCooldownSeconds != nil && (*scaleDownCooldownSeconds < 60 || *scaleDownCooldownSeconds > 3600) {
		return &requestError{status: http.StatusBadRequest, message: "scaleDownCooldownSeconds must be between 60 and 3600"}
	}
	if monitorIntervalMS != nil && (*monitorIntervalMS < 1000 || *monitorIntervalMS > 3600000) {
		return &requestError{status: http.StatusBadRequest, message: "monitorIntervalMs must be between 1000 and 3600000"}
	}
	if inactivityTimeoutSeconds != nil && (*inactivityTimeoutSeconds < 60 || *inactivityTimeoutSeconds > 86400) {
		return &requestError{status: http.StatusBadRequest, message: "inactivityTimeoutSeconds must be between 60 and 86400"}
	}
	if lbStrategy != nil && !isAllowedLBStrategy(*lbStrategy) {
		return &requestError{status: http.StatusBadRequest, message: "lbStrategy must be ROUND_ROBIN or LEAST_CONNECTIONS"}
	}
	return nil
}

func changedTemplateDetails(input updateTemplatePayload) map[string]any {
	details := map[string]any{}
	if input.Name.Present {
		details["name"] = input.Name.Value
	}
	if input.Type.Present {
		if input.Type.Value == nil {
			details["type"] = nil
		} else {
			value := strings.ToUpper(strings.TrimSpace(*input.Type.Value))
			details["type"] = value
		}
	}
	if input.Host.Present {
		details["host"] = input.Host.Value
	}
	if input.Port.Present {
		details["port"] = input.Port.Value
	}
	if input.DeploymentMode.Present {
		if input.DeploymentMode.Value == nil {
			details["deploymentMode"] = nil
		} else {
			details["deploymentMode"] = strings.ToUpper(strings.TrimSpace(*input.DeploymentMode.Value))
		}
	}
	if input.Description.Present {
		details["description"] = input.Description.Value
	}
	if input.APIPort.Present {
		details["apiPort"] = input.APIPort.Value
	}
	if input.AutoScale.Present {
		details["autoScale"] = input.AutoScale.Value
	}
	if input.MinReplicas.Present {
		details["minReplicas"] = input.MinReplicas.Value
	}
	if input.MaxReplicas.Present {
		details["maxReplicas"] = input.MaxReplicas.Value
	}
	if input.SessionsPerInstance.Present {
		details["sessionsPerInstance"] = input.SessionsPerInstance.Value
	}
	if input.ScaleDownCooldownSeconds.Present {
		details["scaleDownCooldownSeconds"] = input.ScaleDownCooldownSeconds.Value
	}
	if input.MonitoringEnabled.Present {
		details["monitoringEnabled"] = input.MonitoringEnabled.Value
	}
	if input.MonitorIntervalMS.Present {
		details["monitorIntervalMs"] = input.MonitorIntervalMS.Value
	}
	if input.InactivityTimeoutSeconds.Present {
		details["inactivityTimeoutSeconds"] = input.InactivityTimeoutSeconds.Value
	}
	if input.PublishPorts.Present {
		details["publishPorts"] = input.PublishPorts.Value
	}
	if input.LBStrategy.Present {
		details["lbStrategy"] = normalizeLBStrategyPtr(input.LBStrategy.Value)
	}
	return details
}

func normalizeLBStrategyPtr(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(*value))
	if normalized == "" {
		return nil
	}
	return &normalized
}

func isAllowedGatewayType(gatewayType string) bool {
	switch gatewayType {
	case "GUACD", "SSH_BASTION", "MANAGED_SSH", "DB_PROXY":
		return true
	default:
		return false
	}
}

func isManagedGatewayType(gatewayType string) bool {
	switch gatewayType {
	case "MANAGED_SSH", "GUACD", "DB_PROXY":
		return true
	default:
		return false
	}
}

func buildTemplateDeploymentName(tenantID, templateName string) string {
	prefix := strings.TrimSpace(tenantID)
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(suffix) > 6 {
		suffix = suffix[:6]
	}
	return fmt.Sprintf("%s-%s-%s", prefix, strings.TrimSpace(templateName), suffix)
}

func isAllowedLBStrategy(strategy string) bool {
	switch strategy {
	case "ROUND_ROBIN", "LEAST_CONNECTIONS":
		return true
	default:
		return false
	}
}
