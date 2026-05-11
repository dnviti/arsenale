package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var profileColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "EMAIL", Field: "email"},
	{Header: "USERNAME", Field: "username"},
	{Header: "ROLE", Field: "role"},
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage your user profile",
}

var profileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get your profile",
	Run:   runProfileGet,
}

var profileUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update your profile",
	Long:  `Update your profile from a JSON/YAML file or with flags: arsenale profile update --username "new-name"`,
	Run:   runProfileUpdate,
}

var profileChangePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "Change your password",
	Long:  `Change password: arsenale profile change-password --current-password <old> --new-password <new>`,
	Run:   runProfileChangePassword,
}

var profileSSHDefaultsCmd = &cobra.Command{
	Use:   "ssh-defaults",
	Short: "Set SSH connection defaults",
	Long:  `Set SSH defaults from a JSON/YAML file: arsenale profile ssh-defaults --from-file ssh.yaml`,
	Run:   runProfileSSHDefaults,
}

var profileRDPDefaultsCmd = &cobra.Command{
	Use:   "rdp-defaults",
	Short: "Set RDP connection defaults",
	Long:  `Set RDP defaults from a JSON/YAML file: arsenale profile rdp-defaults --from-file rdp.yaml`,
	Run:   runProfileRDPDefaults,
}

var profileDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage domain profile",
}

var profileDomainGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get your domain profile",
	Run:   runProfileDomainGet,
}

var profileDomainSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set your domain profile",
	Run:   runProfileDomainSet,
}

var profileDomainClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear your domain profile",
	Run:   runProfileDomainClear,
}

var profileMFACmd = &cobra.Command{
	Use:   "mfa",
	Short: "Manage MFA settings",
}

var profileMFAStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get your MFA status",
	Run:   runProfileMFAStatus,
}

var (
	profileFromFile   string
	profileName       string
	profileCurrentPwd string
	profileNewPwd     string
	profileSSHFile    string
	profileRDPFile    string
	profileDomainFile string
)

func init() {
	rootCmd.AddCommand(profileCmd)

	profileCmd.AddCommand(profileGetCmd)
	profileCmd.AddCommand(profileUpdateCmd)
	profileCmd.AddCommand(profileChangePasswordCmd)
	profileCmd.AddCommand(profileSSHDefaultsCmd)
	profileCmd.AddCommand(profileRDPDefaultsCmd)
	profileCmd.AddCommand(profileDomainCmd)
	profileCmd.AddCommand(profileMFACmd)

	profileDomainCmd.AddCommand(profileDomainGetCmd)
	profileDomainCmd.AddCommand(profileDomainSetCmd)
	profileDomainCmd.AddCommand(profileDomainClearCmd)

	profileMFACmd.AddCommand(profileMFAStatusCmd)

	profileUpdateCmd.Flags().StringVarP(&profileFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	profileUpdateCmd.Flags().StringVar(&profileName, "username", "", "Username")
	profileUpdateCmd.Flags().StringVar(&profileName, "name", "", "Deprecated alias for --username")

	profileChangePasswordCmd.Flags().StringVar(&profileCurrentPwd, "current-password", "", "Current password")
	profileChangePasswordCmd.Flags().StringVar(&profileNewPwd, "new-password", "", "New password")

	profileSSHDefaultsCmd.Flags().StringVarP(&profileSSHFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	profileSSHDefaultsCmd.MarkFlagRequired("from-file")

	profileRDPDefaultsCmd.Flags().StringVarP(&profileRDPFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	profileRDPDefaultsCmd.MarkFlagRequired("from-file")

	profileDomainSetCmd.Flags().StringVarP(&profileDomainFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	profileDomainSetCmd.MarkFlagRequired("from-file")
}

func runProfileGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/user/profile", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, profileColumns)
}

func runProfileUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if profileFromFile != "" {
		data, err = readResourceFromFileOrStdin(profileFromFile)
		if err != nil {
			fatal("%v", err)
		}
		data, err = normalizeProfileUpdatePayload(data)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if profileName == "" {
			fatal("provide --from-file or --username")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"username": profileName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPut("/api/user/profile", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, profileColumns)
}

func runProfileChangePassword(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	currentPwd := profileCurrentPwd
	newPwd := profileNewPwd

	if currentPwd == "" {
		var err error
		currentPwd, err = promptPassword("Current password: ")
		if err != nil {
			fatal("%v", err)
		}
	}

	if newPwd == "" {
		var err error
		newPwd, err = promptPassword("New password: ")
		if err != nil {
			fatal("%v", err)
		}
	}

	payload := map[string]string{
		"oldPassword": currentPwd,
		"newPassword": newPwd,
	}

	body, status, err := apiPut("/api/user/password", payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Password changed successfully")
	}
}

func runProfileSSHDefaults(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(profileSSHFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/user/ssh-defaults", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("SSH defaults updated")
	}
}

func runProfileRDPDefaults(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(profileRDPFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/user/rdp-defaults", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("RDP defaults updated")
	}
}

func runProfileDomainGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/user/domain-profile", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "DOMAIN", Field: "domain"},
		{Header: "USERNAME", Field: "username"},
	})
}

func runProfileDomainSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(profileDomainFile)
	if err != nil {
		fatal("%v", err)
	}
	data, err = normalizeDomainProfilePayload(data)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/user/domain-profile", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Domain profile updated")
	}
}

func runProfileDomainClear(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/user/domain-profile", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Domain profile cleared")
	}
}

func runProfileMFAStatus(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/user/2fa/status", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "ENABLED", Field: "enabled"},
		{Header: "METHOD", Field: "method"},
	})
}

func normalizeProfileUpdatePayload(data []byte) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("profile payload must be a JSON/YAML object: %w", err)
	}
	if raw, ok := payload["name"]; ok {
		if _, exists := payload["username"]; !exists {
			payload["username"] = raw
		}
		delete(payload, "name")
	}
	return json.Marshal(payload)
}

func normalizeDomainProfilePayload(data []byte) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("domain profile payload must be a JSON/YAML object: %w", err)
	}
	aliasField(payload, "domain", "domainName")
	aliasField(payload, "username", "domainUsername")
	aliasField(payload, "password", "domainPassword")
	return json.Marshal(payload)
}

func aliasField(payload map[string]json.RawMessage, oldKey, newKey string) {
	raw, ok := payload[oldKey]
	if !ok {
		return
	}
	if _, exists := payload[newKey]; !exists {
		payload[newKey] = raw
	}
	delete(payload, oldKey)
}
