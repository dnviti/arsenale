package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const sshSandboxRelativePathErrorText = "Only sandbox-relative paths are allowed; remote filesystem browsing is disabled."

var sshFileColumns = []Column{
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "SIZE", Field: "size"},
	{Header: "MODIFIED_AT", Field: "modifiedAt"},
}

var fileSSHCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Manage SSH sandbox file transfers",
}

var fileSSHListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files in the SSH transfer sandbox",
	Run:   runSSHFileList,
}

var fileSSHMkdirCmd = &cobra.Command{
	Use:   "mkdir",
	Short: "Create a directory in the SSH transfer sandbox",
	Run:   runSSHFileMkdir,
}

var fileSSHDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a file or directory in the SSH transfer sandbox",
	Run:   runSSHFileDelete,
}

var fileSSHRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename a file or directory in the SSH transfer sandbox",
	Run:   runSSHFileRename,
}

var fileSSHUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a local file into the SSH transfer sandbox",
	Run:   runSSHFileUpload,
}

var fileSSHDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file from the SSH transfer sandbox",
	Run:   runSSHFileDownload,
}

var (
	sshFileConnection  string
	sshFilePath        string
	sshFileOldPath     string
	sshFileNewPath     string
	sshFileRemotePath  string
	sshFileUploadPath  string
	sshFileDownloadDst string
	sshFileOverrides   credentialOverride
)

func init() {
	fileCmd.AddCommand(fileSSHCmd)
	fileSSHCmd.AddCommand(fileSSHListCmd)
	fileSSHCmd.AddCommand(fileSSHMkdirCmd)
	fileSSHCmd.AddCommand(fileSSHDeleteCmd)
	fileSSHCmd.AddCommand(fileSSHRenameCmd)
	fileSSHCmd.AddCommand(fileSSHUploadCmd)
	fileSSHCmd.AddCommand(fileSSHDownloadCmd)

	for _, subcmd := range []*cobra.Command{
		fileSSHListCmd,
		fileSSHMkdirCmd,
		fileSSHDeleteCmd,
		fileSSHRenameCmd,
		fileSSHUploadCmd,
		fileSSHDownloadCmd,
	} {
		subcmd.Flags().StringVar(&sshFileConnection, "connection", "", "SSH connection name or ID")
		subcmd.MarkFlagRequired("connection")
		addCredentialOverrideFlags(subcmd, &sshFileOverrides)
	}

	fileSSHListCmd.Flags().StringVar(&sshFilePath, "path", ".", "sandbox-relative directory path under workspace/current/ to list; use . for the sandbox root")

	fileSSHMkdirCmd.Flags().StringVar(&sshFilePath, "path", "", "sandbox-relative directory path under workspace/current/ to create")
	fileSSHMkdirCmd.MarkFlagRequired("path")

	fileSSHDeleteCmd.Flags().StringVar(&sshFilePath, "path", "", "sandbox-relative file or directory path under workspace/current/ to delete")
	fileSSHDeleteCmd.MarkFlagRequired("path")

	fileSSHRenameCmd.Flags().StringVar(&sshFileOldPath, "from", "", "existing sandbox-relative path under workspace/current/")
	fileSSHRenameCmd.Flags().StringVar(&sshFileNewPath, "to", "", "new sandbox-relative path under workspace/current/")
	fileSSHRenameCmd.MarkFlagRequired("from")
	fileSSHRenameCmd.MarkFlagRequired("to")

	fileSSHUploadCmd.Flags().StringVar(&sshFileUploadPath, "file", "", "Local file to upload")
	fileSSHUploadCmd.Flags().StringVar(&sshFileRemotePath, "to", "", "sandbox-relative destination path under workspace/current/")
	fileSSHUploadCmd.Flags().StringVar(&sshFileRemotePath, "remote-path", "", "deprecated alias for --to")
	_ = fileSSHUploadCmd.Flags().MarkHidden("remote-path")
	fileSSHUploadCmd.MarkFlagRequired("file")

	fileSSHDownloadCmd.Flags().StringVar(&sshFilePath, "path", "", "sandbox-relative file path under workspace/current/ to download")
	fileSSHDownloadCmd.Flags().StringVar(&sshFileDownloadDst, "dest", ".", "Destination file or directory")
	fileSSHDownloadCmd.MarkFlagRequired("path")
}

func runSSHFileList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanPath, err := normalizeSSHSandboxCLIPath(sshFilePath, true)
	if err != nil {
		fatal("%v", err)
	}
	body["path"] = cleanPath

	respBody, status, err := apiPost("/api/files/ssh/list", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)
	if err := printer().Print(extractWrappedJSONField(respBody, "entries"), sshFileColumns); err != nil {
		fatal("%v", err)
	}
}

func runSSHFileMkdir(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanPath, err := normalizeSSHSandboxCLIPath(sshFilePath, false)
	if err != nil {
		fatal("%v", err)
	}
	body["path"] = cleanPath

	respBody, status, err := apiPost("/api/files/ssh/mkdir", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)
	if !quiet {
		fmt.Printf("Directory %q created\n", cleanPath)
	}
}

func runSSHFileDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanPath, err := normalizeSSHSandboxCLIPath(sshFilePath, false)
	if err != nil {
		fatal("%v", err)
	}
	body["path"] = cleanPath

	respBody, status, err := apiPost("/api/files/ssh/delete", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)
	if !quiet {
		fmt.Printf("Sandbox path %q deleted\n", cleanPath)
	}
}

func runSSHFileRename(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanOldPath, err := normalizeSSHSandboxCLIPath(sshFileOldPath, false)
	if err != nil {
		fatal("%v", err)
	}
	cleanNewPath, err := normalizeSSHSandboxCLIPath(sshFileNewPath, false)
	if err != nil {
		fatal("%v", err)
	}
	body["oldPath"] = cleanOldPath
	body["newPath"] = cleanNewPath

	respBody, status, err := apiPost("/api/files/ssh/rename", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)
	if !quiet {
		fmt.Printf("Sandbox path %q renamed to %q\n", cleanOldPath, cleanNewPath)
	}
}

func runSSHFileUpload(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanRemotePath, err := normalizeSSHSandboxCLIPath(sshFileRemotePath, false)
	if err != nil {
		fatal("%v", err)
	}

	fields := map[string]string{
		"connectionId": body["connectionId"].(string),
		"remotePath":   cleanRemotePath,
	}
	if value := strings.TrimSpace(sshFileOverrides.Username); value != "" {
		fields["username"] = value
	}
	if value := strings.TrimSpace(sshFileOverrides.Password); value != "" {
		fields["password"] = value
	}
	if value := strings.TrimSpace(sshFileOverrides.Domain); value != "" {
		fields["domain"] = value
	}
	if value := strings.TrimSpace(sshFileOverrides.CredentialMode); value != "" {
		fields["credentialMode"] = value
	}

	respBody, status, err := apiUploadWithFields("/api/files/ssh/upload", sshFileUploadPath, fields, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)
	if err := printer().PrintCreated(respBody, "name"); err != nil {
		fatal("%v", err)
	}
}

func runSSHFileDownload(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, body := buildConnectionCredentialBody(sshFileConnection, cfg, sshFileOverrides)
	if conn.Type != "SSH" {
		fatal("connection %q is type %s, not SSH", conn.Name, conn.Type)
	}
	cleanPath, err := normalizeSSHSandboxCLIPath(sshFilePath, false)
	if err != nil {
		fatal("%v", err)
	}
	body["path"] = cleanPath

	respBody, status, err := apiPost("/api/files/ssh/download", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)

	destPath := resolveSSHDownloadDestination(cleanPath, strings.TrimSpace(sshFileDownloadDst))
	if err := writeBytesToPath(destPath, respBody); err != nil {
		fatal("%v", err)
	}
	if !quiet {
		fmt.Printf("Downloaded %q to %s\n", cleanPath, destPath)
	}
}

func resolveSSHDownloadDestination(remotePath, destination string) string {
	if destination == "" {
		destination = "."
	}
	if stat, err := os.Stat(destination); err == nil && stat.IsDir() {
		return filepath.Join(destination, path.Base(remotePath))
	}
	if strings.HasSuffix(destination, string(os.PathSeparator)) {
		return filepath.Join(destination, path.Base(remotePath))
	}
	return destination
}

func normalizeSSHSandboxCLIPath(input string, allowRoot bool) (string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		if allowRoot {
			return ".", nil
		}
		return "", errors.New("path is required")
	}
	if raw == "/" || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") || strings.Contains(raw, "://") || strings.HasPrefix(strings.ToLower(raw), "file:") {
		return "", errors.New(sshSandboxRelativePathErrorText)
	}
	if len(raw) >= 2 && ((raw[0] >= 'a' && raw[0] <= 'z') || (raw[0] >= 'A' && raw[0] <= 'Z')) && raw[1] == ':' {
		return "", errors.New(sshSandboxRelativePathErrorText)
	}
	for _, segment := range strings.Split(strings.ReplaceAll(raw, "\\", "/"), "/") {
		if segment == ".." {
			return "", errors.New(sshSandboxRelativePathErrorText)
		}
	}
	clean := path.Clean(strings.TrimPrefix(raw, "./"))
	if clean == "." {
		if allowRoot {
			return ".", nil
		}
		return "", errors.New(sshSandboxRelativePathErrorText)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", errors.New(sshSandboxRelativePathErrorText)
	}
	return clean, nil
}
