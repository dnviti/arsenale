package main

import (
	"github.com/dnviti/arsenale/backend/internal/accesspolicies"
	"github.com/dnviti/arsenale/backend/internal/adminapi"
	"github.com/dnviti/arsenale/backend/internal/auditapi"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/authservice"
	"github.com/dnviti/arsenale/backend/internal/checkouts"
	"github.com/dnviti/arsenale/backend/internal/cliapi"
	"github.com/dnviti/arsenale/backend/internal/connections"
	"github.com/dnviti/arsenale/backend/internal/dbauditapi"
	"github.com/dnviti/arsenale/backend/internal/dbsessions"
	"github.com/dnviti/arsenale/backend/internal/desktopsessions"
	"github.com/dnviti/arsenale/backend/internal/externalvaultapi"
	"github.com/dnviti/arsenale/backend/internal/files"
	"github.com/dnviti/arsenale/backend/internal/folders"
	"github.com/dnviti/arsenale/backend/internal/gateways"
	"github.com/dnviti/arsenale/backend/internal/geoipapi"
	"github.com/dnviti/arsenale/backend/internal/importexportapi"
	"github.com/dnviti/arsenale/backend/internal/keystrokepolicies"
	"github.com/dnviti/arsenale/backend/internal/ldapapi"
	"github.com/dnviti/arsenale/backend/internal/mfaapi"
	"github.com/dnviti/arsenale/backend/internal/modelgatewayapi"
	"github.com/dnviti/arsenale/backend/internal/notifications"
	"github.com/dnviti/arsenale/backend/internal/oauthapi"
	"github.com/dnviti/arsenale/backend/internal/orchestration"
	"github.com/dnviti/arsenale/backend/internal/passwordrotationapi"
	"github.com/dnviti/arsenale/backend/internal/publicconfig"
	"github.com/dnviti/arsenale/backend/internal/publicshareapi"
	"github.com/dnviti/arsenale/backend/internal/rdgatewayapi"
	"github.com/dnviti/arsenale/backend/internal/recordingsapi"
	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
	"github.com/dnviti/arsenale/backend/internal/secretsmeta"
	"github.com/dnviti/arsenale/backend/internal/sessionadmin"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/internal/setup"
	"github.com/dnviti/arsenale/backend/internal/sshproxyapi"
	"github.com/dnviti/arsenale/backend/internal/sshsessions"
	"github.com/dnviti/arsenale/backend/internal/syncprofiles"
	"github.com/dnviti/arsenale/backend/internal/systemsettingsapi"
	"github.com/dnviti/arsenale/backend/internal/tabs"
	"github.com/dnviti/arsenale/backend/internal/teams"
	"github.com/dnviti/arsenale/backend/internal/tenants"
	"github.com/dnviti/arsenale/backend/internal/tenantvaultapi"
	"github.com/dnviti/arsenale/backend/internal/users"
	"github.com/dnviti/arsenale/backend/internal/vaultapi"
	"github.com/dnviti/arsenale/backend/internal/vaultfolders"
	"github.com/jackc/pgx/v5/pgxpool"
)

type apiDependencies struct {
	db                      *pgxpool.Pool
	store                   *orchestration.Store
	sessionStore            *sessions.Store
	desktopSessionService   desktopsessions.Service
	databaseSessionService  dbsessions.Service
	setupService            setup.Service
	publicConfigService     publicconfig.Service
	publicShareService      publicshareapi.Service
	authService             authservice.Service
	mfaService              mfaapi.Service
	userService             users.Service
	connectionService       connections.Service
	importExportService     importexportapi.Service
	cliService              cliapi.Service
	checkoutService         checkouts.Service
	folderService           folders.Service
	vaultFolderService      vaultfolders.Service
	fileService             files.Service
	gatewayService          gateways.Service
	notificationService     notifications.Service
	oauthService            oauthapi.Service
	passwordRotationService passwordrotationapi.Service
	geoIPService            *geoipapi.Service
	ldapService             ldapapi.Service
	rdGatewayService        rdgatewayapi.Service
	recordingService        recordingsapi.Service
	secretsMetaService      secretsmeta.Service
	tenantVaultService      tenantvaultapi.Service
	tabsService             tabs.Service
	tenantService           tenants.Service
	teamService             teams.Service
	syncProfileService      syncprofiles.Service
	externalVaultService    externalvaultapi.Service
	vaultService            vaultapi.Service
	adminService            adminapi.Service
	systemSettingsService   systemsettingsapi.Service
	auditService            auditapi.Service
	dbAuditService          dbauditapi.Service
	accessPolicyService     accesspolicies.Service
	keystrokePolicyService  keystrokepolicies.Service
	sessionAdminService     sessionadmin.Service
	sshSessionService       sshsessions.Service
	sshProxyService         sshproxyapi.Service
	modelGatewayService     modelgatewayapi.Service
	authenticator           *authn.Authenticator
	features                runtimefeatures.Manifest
}
