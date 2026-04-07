package gateways

import "time"

type gatewayTemplateResponse struct {
	ID                       string               `json:"id"`
	Name                     string               `json:"name"`
	Type                     string               `json:"type"`
	Host                     string               `json:"host"`
	Port                     int                  `json:"port"`
	DeploymentMode           string               `json:"deploymentMode"`
	Description              *string              `json:"description"`
	APIPort                  *int                 `json:"apiPort"`
	AutoScale                bool                 `json:"autoScale"`
	MinReplicas              int                  `json:"minReplicas"`
	MaxReplicas              int                  `json:"maxReplicas"`
	SessionsPerInstance      int                  `json:"sessionsPerInstance"`
	ScaleDownCooldownSeconds int                  `json:"scaleDownCooldownSeconds"`
	MonitoringEnabled        bool                 `json:"monitoringEnabled"`
	MonitorIntervalMS        int                  `json:"monitorIntervalMs"`
	InactivityTimeoutSeconds int                  `json:"inactivityTimeoutSeconds"`
	PublishPorts             bool                 `json:"publishPorts"`
	LBStrategy               string               `json:"lbStrategy"`
	TenantID                 string               `json:"tenantId"`
	CreatedByID              string               `json:"createdById"`
	CreatedAt                time.Time            `json:"createdAt"`
	UpdatedAt                time.Time            `json:"updatedAt"`
	Count                    gatewayTemplateCount `json:"_count"`
}

type gatewayTemplateCount struct {
	Gateways int `json:"gateways"`
}

type createTemplatePayload struct {
	Name                     string  `json:"name"`
	Type                     string  `json:"type"`
	Host                     *string `json:"host"`
	Port                     *int    `json:"port"`
	DeploymentMode           *string `json:"deploymentMode"`
	Description              *string `json:"description"`
	APIPort                  *int    `json:"apiPort"`
	AutoScale                *bool   `json:"autoScale"`
	MinReplicas              *int    `json:"minReplicas"`
	MaxReplicas              *int    `json:"maxReplicas"`
	SessionsPerInstance      *int    `json:"sessionsPerInstance"`
	ScaleDownCooldownSeconds *int    `json:"scaleDownCooldownSeconds"`
	MonitoringEnabled        *bool   `json:"monitoringEnabled"`
	MonitorIntervalMS        *int    `json:"monitorIntervalMs"`
	InactivityTimeoutSeconds *int    `json:"inactivityTimeoutSeconds"`
	PublishPorts             *bool   `json:"publishPorts"`
	LBStrategy               *string `json:"lbStrategy"`
}

type normalizedCreateTemplatePayload struct {
	Name                     string
	Type                     string
	Host                     string
	Port                     int
	DeploymentMode           string
	Description              *string
	APIPort                  *int
	AutoScale                *bool
	MinReplicas              *int
	MaxReplicas              *int
	SessionsPerInstance      *int
	ScaleDownCooldownSeconds *int
	MonitoringEnabled        *bool
	MonitorIntervalMS        *int
	InactivityTimeoutSeconds *int
	PublishPorts             *bool
	LBStrategy               *string
}

type updateTemplatePayload struct {
	Name                     optionalString `json:"name"`
	Type                     optionalString `json:"type"`
	Host                     optionalString `json:"host"`
	Port                     optionalInt    `json:"port"`
	DeploymentMode           optionalString `json:"deploymentMode"`
	Description              optionalString `json:"description"`
	APIPort                  optionalInt    `json:"apiPort"`
	AutoScale                optionalBool   `json:"autoScale"`
	MinReplicas              optionalInt    `json:"minReplicas"`
	MaxReplicas              optionalInt    `json:"maxReplicas"`
	SessionsPerInstance      optionalInt    `json:"sessionsPerInstance"`
	ScaleDownCooldownSeconds optionalInt    `json:"scaleDownCooldownSeconds"`
	MonitoringEnabled        optionalBool   `json:"monitoringEnabled"`
	MonitorIntervalMS        optionalInt    `json:"monitorIntervalMs"`
	InactivityTimeoutSeconds optionalInt    `json:"inactivityTimeoutSeconds"`
	PublishPorts             optionalBool   `json:"publishPorts"`
	LBStrategy               optionalString `json:"lbStrategy"`
}

type gatewayTemplateRecord struct {
	ID                       string
	Name                     string
	Type                     string
	Host                     string
	Port                     int
	DeploymentMode           string
	Description              *string
	APIPort                  *int
	AutoScale                bool
	MinReplicas              int
	MaxReplicas              int
	SessionsPerInstance      int
	ScaleDownCooldownSeconds int
	MonitoringEnabled        bool
	MonitorIntervalMS        int
	InactivityTimeoutSeconds int
	PublishPorts             bool
	LBStrategy               string
	TenantID                 string
	CreatedByID              string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	GatewayCount             int
}

const gatewayTemplateSelect = `
SELECT
	t.id,
	t.name,
	t.type::text,
	t.host,
	t.port,
	t."deploymentMode"::text,
	t.description,
	t."apiPort",
	t."autoScale",
	t."minReplicas",
	t."maxReplicas",
	t."sessionsPerInstance",
	t."scaleDownCooldownSeconds",
	t."monitoringEnabled",
	t."monitorIntervalMs",
	t."inactivityTimeoutSeconds",
	t."publishPorts",
	t."lbStrategy"::text,
	t."tenantId",
	t."createdById",
	t."createdAt",
	t."updatedAt",
	COALESCE(gateway_counts.count, 0) AS "gatewayCount"
FROM "GatewayTemplate" t
LEFT JOIN LATERAL (
	SELECT COUNT(*)::int AS count
	FROM "Gateway" g
	WHERE g."templateId" = t.id
) gateway_counts ON true
`

const insertGatewayTemplateSQL = `
INSERT INTO "GatewayTemplate" (
	id,
	name,
	type,
	host,
	port,
	"deploymentMode",
	description,
	"apiPort",
	"autoScale",
	"minReplicas",
	"maxReplicas",
	"sessionsPerInstance",
	"scaleDownCooldownSeconds",
	"monitoringEnabled",
	"monitorIntervalMs",
	"inactivityTimeoutSeconds",
	"publishPorts",
	"lbStrategy",
	"tenantId",
	"createdById",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1,
	$2,
	$3::"GatewayType",
	$4,
	$5,
	$6::"GatewayDeploymentMode",
	$7,
	$8,
	COALESCE($9, false),
	COALESCE($10, 1),
	COALESCE($11, 5),
	COALESCE($12, 10),
	COALESCE($13, 300),
	COALESCE($14, true),
	COALESCE($15, 5000),
	COALESCE($16, 3600),
	COALESCE($17, false),
	COALESCE($18::"LoadBalancingStrategy", 'ROUND_ROBIN'::"LoadBalancingStrategy"),
	$19,
	$20,
	NOW(),
	NOW()
)
`

const insertGatewayFromTemplateSQL = `
INSERT INTO "Gateway" (
	id,
	name,
	type,
	host,
	port,
	"deploymentMode",
	description,
	"apiPort",
	"isManaged",
	"desiredReplicas",
	"autoScale",
	"minReplicas",
	"maxReplicas",
	"sessionsPerInstance",
	"scaleDownCooldownSeconds",
	"monitoringEnabled",
	"monitorIntervalMs",
	"inactivityTimeoutSeconds",
	"publishPorts",
	"lbStrategy",
	"tenantId",
	"createdById",
	"templateId",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1,
	$2,
	$3::"GatewayType",
	$4,
	$5,
	$6::"GatewayDeploymentMode",
	$7,
	$8,
	$9,
	$10,
	$11,
	$12,
	$13,
	$14,
	$15,
	$16,
	$17,
	$18,
	$19,
	$20::"LoadBalancingStrategy",
	$21,
	$22,
	$23,
	NOW(),
	NOW()
)
`
