package passwordrotationapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"
)

const (
	rotationSSHTimeout = 15 * time.Second
	passwordLength     = 32
	passwordCharset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?"
	lowerCharset       = "abcdefghijklmnopqrstuvwxyz"
	upperCharset       = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitCharset       = "0123456789"
	specialCharset     = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

type rotationConnection struct {
	ID   string
	Type string
	Host string
	Port int
}

type loginPayload struct {
	Username string
	Password string
	Data     map[string]any
}

func (s Service) TriggerRotation(ctx context.Context, userID, tenantID, secretID, trigger, ipAddress string) (rotationResult, error) {
	secret, err := s.requireManageAccess(ctx, userID, tenantID, secretID)
	if err != nil {
		return rotationResult{}, err
	}
	if secret.Type != "LOGIN" {
		return rotationResult{}, &requestError{status: 400, message: "Password rotation is only supported for LOGIN-type secrets"}
	}

	connection, err := s.loadRotationConnection(ctx, secretID)
	if err != nil {
		return rotationResult{}, err
	}

	payload, err := s.Resolver.LoadSecretPayload(ctx, userID, secretID, tenantID)
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) && reqErr.status == 404 {
			return rotationResult{}, &requestError{status: 404, message: "Secret not found"}
		}
		return rotationResult{}, err
	}

	login, err := parseLoginPayload(payload)
	if err != nil {
		return rotationResult{}, &requestError{status: 400, message: err.Error()}
	}

	targetOS := detectTargetOS(connection.Type)
	newPassword, err := generateStrongPassword(passwordLength)
	if err != nil {
		return rotationResult{}, fmt.Errorf("generate rotated password: %w", err)
	}

	logID, err := s.createRotationLog(ctx, secretID, trigger, targetOS, connection.Host, login.Username, userID)
	if err != nil {
		return rotationResult{}, err
	}

	result := rotationResult{
		Success:  false,
		SecretID: secretID,
		LogID:    logID,
	}
	startedAt := time.Now()
	durationMs := func() int {
		return int(time.Since(startedAt).Milliseconds())
	}

	changeErr := s.changeRemotePassword(connection, targetOS, login.Username, login.Password, newPassword)
	if changeErr != nil {
		errorMessage := changeErr.Error()
		result.Error = errorMessage
		if err := s.markRotationFailed(ctx, logID, errorMessage, durationMs()); err != nil {
			return rotationResult{}, err
		}
		_ = s.insertAuditAction(ctx, userID, "PASSWORD_ROTATION_FAILED", secretID, map[string]any{
			"trigger":    trigger,
			"targetOS":   targetOS,
			"targetHost": connection.Host,
			"targetUser": login.Username,
			"error":      errorMessage,
		}, ipAddress)
		return result, nil
	}

	updatedPayload, err := login.withPassword(newPassword)
	if err != nil {
		return rotationResult{}, err
	}
	ciphertext, iv, tag, err := s.Resolver.EncryptPayloadForScope(ctx, userID, secret.Scope, secret.TeamID, secret.TenantID, updatedPayload)
	if err != nil {
		return rotationResult{}, err
	}

	if err := s.persistSuccessfulRotation(ctx, userID, secretID, logID, trigger, ciphertext, iv, tag, durationMs()); err != nil {
		return rotationResult{}, err
	}

	result.Success = true
	result.Error = ""
	_ = s.insertAuditAction(ctx, userID, "PASSWORD_ROTATION_SUCCESS", secretID, map[string]any{
		"trigger":    trigger,
		"targetOS":   targetOS,
		"targetHost": connection.Host,
		"targetUser": login.Username,
		"durationMs": durationMs(),
	}, ipAddress)
	return result, nil
}

func (s Service) loadRotationConnection(ctx context.Context, secretID string) (rotationConnection, error) {
	if s.DB == nil {
		return rotationConnection{}, errors.New("database is unavailable")
	}

	var connection rotationConnection
	err := s.DB.QueryRow(ctx, `
SELECT id, type::text, host, port
FROM "Connection"
WHERE "credentialSecretId" = $1
ORDER BY "createdAt" ASC, id ASC
LIMIT 1
`, secretID).Scan(&connection.ID, &connection.Type, &connection.Host, &connection.Port)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rotationConnection{}, &requestError{status: 400, message: "Secret must be linked to at least one connection for rotation"}
		}
		return rotationConnection{}, fmt.Errorf("load rotation connection: %w", err)
	}
	return connection, nil
}

func (s Service) createRotationLog(ctx context.Context, secretID, trigger, targetOS, targetHost, targetUser, userID string) (string, error) {
	if s.DB == nil {
		return "", errors.New("database is unavailable")
	}
	logID := uuid.NewString()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "PasswordRotationLog" (
	id,
	"secretId",
	status,
	trigger,
	"targetOS",
	"targetHost",
	"targetUser",
	"initiatedBy"
) VALUES (
	$1,
	$2,
	'PENDING'::"RotationStatus",
	$3::"RotationTrigger",
	$4::"RotationTargetOS",
	$5,
	$6,
	$7
)
`, logID, secretID, strings.ToUpper(strings.TrimSpace(trigger)), strings.ToUpper(strings.TrimSpace(targetOS)), targetHost, targetUser, userID); err != nil {
		return "", fmt.Errorf("insert password rotation log: %w", err)
	}
	return logID, nil
}

func (s Service) markRotationFailed(ctx context.Context, logID, errorMessage string, durationMs int) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	if _, err := s.DB.Exec(ctx, `
UPDATE "PasswordRotationLog"
SET status = 'FAILED'::"RotationStatus",
    "errorMessage" = $2,
    "durationMs" = $3
WHERE id = $1
`, logID, errorMessage, nullableInt(durationMs)); err != nil {
		return fmt.Errorf("update failed password rotation log: %w", err)
	}
	return nil
}

func (s Service) persistSuccessfulRotation(ctx context.Context, userID, secretID, logID, trigger, ciphertext, iv, tag string, durationMs int) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password rotation update: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentVersion int
	if err := tx.QueryRow(ctx, `
SELECT "currentVersion"
FROM "VaultSecret"
WHERE id = $1
FOR UPDATE
`, secretID).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: 404, message: "Secret not found"}
		}
		return fmt.Errorf("lock vault secret for password rotation: %w", err)
	}

	newVersion := currentVersion + 1
	if _, err := tx.Exec(ctx, `
UPDATE "VaultSecret"
SET "encryptedData" = $2,
    "dataIV" = $3,
    "dataTag" = $4,
    "currentVersion" = $5,
    "lastRotatedAt" = NOW(),
    "updatedAt" = NOW()
WHERE id = $1
`, secretID, ciphertext, iv, tag, newVersion); err != nil {
		return fmt.Errorf("update rotated vault secret: %w", err)
	}

	changeNote := fmt.Sprintf("Password rotated (%s)", strings.ToLower(strings.TrimSpace(trigger)))
	if _, err := tx.Exec(ctx, `
INSERT INTO "VaultSecretVersion" (
	id,
	"secretId",
	version,
	"encryptedData",
	"dataIV",
	"dataTag",
	"changedBy",
	"changeNote"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, uuid.NewString(), secretID, newVersion, ciphertext, iv, tag, userID, changeNote); err != nil {
		return fmt.Errorf("insert rotated vault secret version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE "PasswordRotationLog"
SET status = 'SUCCESS'::"RotationStatus",
    "durationMs" = $2
WHERE id = $1
`, logID, nullableInt(durationMs)); err != nil {
		return fmt.Errorf("mark password rotation log successful: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password rotation update: %w", err)
	}
	return nil
}

func parseLoginPayload(payload json.RawMessage) (loginPayload, error) {
	if !json.Valid(payload) {
		return loginPayload{}, errors.New("Secret data is not LOGIN type")
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return loginPayload{}, fmt.Errorf("decode secret payload: %w", err)
	}

	secretType := strings.ToUpper(strings.TrimSpace(stringValue(data["type"])))
	if secretType != "LOGIN" {
		return loginPayload{}, errors.New("Secret data is not LOGIN type")
	}

	username := strings.TrimSpace(stringValue(data["username"]))
	if username == "" {
		return loginPayload{}, errors.New("Secret login username is required for password rotation")
	}

	password := stringValue(data["password"])
	if password == "" {
		return loginPayload{}, errors.New("Secret login password is required for password rotation")
	}

	return loginPayload{
		Username: username,
		Password: password,
		Data:     data,
	}, nil
}

func (p loginPayload) withPassword(newPassword string) (json.RawMessage, error) {
	updated := make(map[string]any, len(p.Data))
	for key, value := range p.Data {
		updated[key] = value
	}
	updated["password"] = newPassword
	updated["type"] = "LOGIN"

	raw, err := json.Marshal(updated)
	if err != nil {
		return nil, fmt.Errorf("encode updated secret payload: %w", err)
	}
	return raw, nil
}

func detectTargetOS(connectionType string) string {
	if strings.EqualFold(strings.TrimSpace(connectionType), "RDP") {
		return "WINDOWS"
	}
	return "LINUX"
}

func (s Service) changeRemotePassword(connection rotationConnection, targetOS, username, currentPassword, newPassword string) error {
	switch strings.ToUpper(strings.TrimSpace(targetOS)) {
	case "WINDOWS":
		return changePasswordViaWindowsSSH(connection.Host, connection.Port, username, currentPassword, newPassword)
	default:
		return changePasswordViaSSH(connection.Host, connection.Port, username, currentPassword, newPassword)
	}
}

func changePasswordViaSSH(host string, port int, username, currentPassword, newPassword string) error {
	cmd := fmt.Sprintf("echo '%s:%s' | sudo chpasswd", escapePOSIXSingleQuoted(username), escapePOSIXSingleQuoted(newPassword))
	return runSSHCommand(host, port, username, currentPassword, cmd, "chpasswd")
}

func changePasswordViaWindowsSSH(host string, port int, username, currentPassword, newPassword string) error {
	cmd := fmt.Sprintf(
		`powershell -Command "Set-LocalUser -Name '%s' -Password (ConvertTo-SecureString '%s' -AsPlainText -Force)"`,
		escapePowerShellSingleQuoted(username),
		escapePowerShellSingleQuoted(newPassword),
	)
	return runSSHCommand(host, port, username, currentPassword, cmd, "PowerShell Set-LocalUser")
}

func runSSHCommand(host string, port int, username, currentPassword, command, label string) error {
	address := net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
	client, err := ssh.Dial("tcp", address, &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(currentPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         rotationSSHTimeout,
	})
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("SSH exec failed: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err == nil {
		return nil
	}

	stderr := strings.TrimSpace(string(output))
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		if stderr != "" {
			return fmt.Errorf("%s exited with code %d: %s", label, exitErr.ExitStatus(), stderr)
		}
		return fmt.Errorf("%s exited with code %d", label, exitErr.ExitStatus())
	}
	if stderr != "" {
		return fmt.Errorf("SSH exec failed: %s", stderr)
	}
	return fmt.Errorf("SSH exec failed: %w", err)
}

func generateStrongPassword(length int) (string, error) {
	if length < 4 {
		length = 4
	}

	chars := make([]byte, length)
	for i := range chars {
		index, err := uniformRandom(len(passwordCharset))
		if err != nil {
			return "", err
		}
		chars[i] = passwordCharset[index]
	}

	requiredSets := []string{lowerCharset, upperCharset, digitCharset, specialCharset}
	for i, charset := range requiredSets {
		index, err := uniformRandom(len(charset))
		if err != nil {
			return "", err
		}
		chars[i] = charset[index]
	}

	for i := len(chars) - 1; i > 0; i-- {
		j, err := uniformRandom(i + 1)
		if err != nil {
			return "", err
		}
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars), nil
}

func uniformRandom(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("read random index: %w", err)
	}
	return int(n.Int64()), nil
}

func escapePOSIXSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", `'\''`)
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func nullableInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}
