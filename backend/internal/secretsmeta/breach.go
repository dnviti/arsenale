package secretsmeta

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
)

const hibpAPIURL = "https://api.pwnedpasswords.com/range/"
const hibpUserAgent = "Arsenale-PasswordCheck"
const hibpTimeout = 5 * time.Second

type breachCheckResponse struct {
	PwnedCount int `json:"pwnedCount"`
}

type batchBreachCheckResponse struct {
	Checked int `json:"checked"`
	Pwned   int `json:"pwned"`
	Results []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PwnedCount int    `json:"pwnedCount"`
	} `json:"results"`
}

type secretsRequestError struct {
	status  int
	message string
}

func (e *secretsRequestError) Error() string {
	return e.message
}

func (s Service) HandleCheckBreach(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.CheckSecretBreach(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleCheckAllBreaches(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.CheckAllSecretBreaches(r.Context(), claims.UserID, claims.TenantID)
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) CheckSecretBreach(ctx context.Context, userID, tenantID, secretID string) (breachCheckResponse, error) {
	detail, err := s.LoadSecret(ctx, userID, tenantID, secretID)
	if err != nil {
		return breachCheckResponse{}, err
	}

	password := extractPasswordFromPayload(detail.Data)
	if password == "" {
		return breachCheckResponse{PwnedCount: 0}, nil
	}

	pwnedCount, err := checkPwnedPassword(ctx, password)
	if err != nil {
		return breachCheckResponse{}, err
	}

	if !detail.Shared {
		if err := s.updateSecretPwnedCount(ctx, detail.ID, pwnedCount); err != nil {
			return breachCheckResponse{}, err
		}
	}

	return breachCheckResponse{PwnedCount: pwnedCount}, nil
}

func (s Service) CheckAllSecretBreaches(ctx context.Context, userID, tenantID string) (batchBreachCheckResponse, error) {
	items, err := s.LoadList(ctx, userID, tenantID, listFilters{})
	if err != nil {
		return batchBreachCheckResponse{}, err
	}

	secretsToCheck := make([]secretListItem, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "LOGIN", "SSH_KEY", "CERTIFICATE":
			secretsToCheck = append(secretsToCheck, item)
		}
	}

	var (
		result       batchBreachCheckResponse
		mu           sync.Mutex
		currentIndex int
	)

	worker := func() {
		for {
			mu.Lock()
			if currentIndex >= len(secretsToCheck) {
				mu.Unlock()
				return
			}
			item := secretsToCheck[currentIndex]
			currentIndex++
			mu.Unlock()

			detail, err := s.LoadSecret(ctx, userID, tenantID, item.ID)
			if err != nil {
				continue
			}

			password := extractPasswordFromPayload(detail.Data)
			if password == "" {
				continue
			}

			pwnedCount, err := checkPwnedPassword(ctx, password)
			if err != nil {
				continue
			}

			if !detail.Shared {
				_ = s.updateSecretPwnedCount(ctx, detail.ID, pwnedCount)
			}

			mu.Lock()
			result.Checked++
			if pwnedCount > 0 {
				result.Pwned++
				result.Results = append(result.Results, struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					PwnedCount int    `json:"pwnedCount"`
				}{
					ID:         item.ID,
					Name:       item.Name,
					PwnedCount: pwnedCount,
				})
			}
			mu.Unlock()
		}
	}

	workers := 5
	if len(secretsToCheck) < workers {
		workers = len(secretsToCheck)
	}
	if workers == 0 {
		workers = 1
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			worker()
		}()
	}
	wg.Wait()

	return result, nil
}

func extractPasswordFromPayload(raw json.RawMessage) string {
	var payload struct {
		Type       string `json:"type"`
		Password   string `json:"password"`
		Passphrase string `json:"passphrase"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}

	switch strings.ToUpper(strings.TrimSpace(payload.Type)) {
	case "LOGIN":
		return strings.TrimSpace(payload.Password)
	case "SSH_KEY", "CERTIFICATE":
		return strings.TrimSpace(payload.Passphrase)
	default:
		return ""
	}
}

func checkPwnedPassword(ctx context.Context, password string) (int, error) {
	sum := sha1.Sum([]byte(password))
	sha1Hex := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix := sha1Hex[:5]
	suffix := sha1Hex[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hibpAPIURL+prefix, nil)
	if err != nil {
		return 0, fmt.Errorf("prepare hibp request: %w", err)
	}
	req.Header.Set("User-Agent", hibpUserAgent)
	req.Header.Set("Add-Padding", "true")

	client := &http.Client{Timeout: hibpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		if hibpFailOpenEnabled() {
			return 0, nil
		}
		return 0, &secretsRequestError{status: http.StatusServiceUnavailable, message: "Password strength could not be verified. Please try again later."}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if hibpFailOpenEnabled() {
			return 0, nil
		}
		return 0, &secretsRequestError{status: http.StatusServiceUnavailable, message: "Password strength could not be verified. Please try again later."}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if hibpFailOpenEnabled() {
			return 0, nil
		}
		return 0, &secretsRequestError{status: http.StatusServiceUnavailable, message: "Password strength could not be verified. Please try again later."}
	}

	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		hashSuffix, countText, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found || hashSuffix != suffix {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(countText))
		if err != nil {
			return 0, nil
		}
		return count, nil
	}

	return 0, nil
}

func hibpFailOpenEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("HIBP_FAIL_OPEN")), "true")
}

func (s Service) updateSecretPwnedCount(ctx context.Context, secretID string, pwnedCount int) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}
	if _, err := s.DB.Exec(ctx, `UPDATE "VaultSecret" SET "pwnedCount" = $2 WHERE id = $1`, secretID, pwnedCount); err != nil {
		return fmt.Errorf("update secret breach count: %w", err)
	}
	return nil
}

func (s Service) handleSecretsError(w http.ResponseWriter, err error) {
	var resolverErr *credentialresolver.RequestError
	if errors.As(err, &resolverErr) {
		app.ErrorJSON(w, resolverErr.Status, resolverErr.Message)
		return
	}

	var reqErr *secretsRequestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}

	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}
