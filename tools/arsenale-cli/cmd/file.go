package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var fileColumns = []Column{
	{Header: "NAME", Field: "name"},
	{Header: "SIZE", Field: "size"},
	{Header: "MODIFIED_AT", Field: "modifiedAt"},
}

var fileHistoryColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "FILE_NAME", Field: "fileName"},
	{Header: "RESTORED_NAME", Field: "restoredName"},
	{Header: "PROTOCOL", Field: "protocol"},
	{Header: "SIZE", Field: "size"},
	{Header: "TRANSFER_AT", Field: "transferAt"},
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage sandbox file transfers",
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all files",
	Run:   runFileList,
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file",
	Long:  `Upload a file: arsenale file upload --file /path/to/file.txt`,
	Run:   runFileUpload,
}

var fileDownloadCmd = &cobra.Command{
	Use:   "download <name>",
	Short: "Download a file",
	Long:  `Download a file: arsenale file download myfile.txt --dest /tmp`,
	Args:  cobra.ExactArgs(1),
	Run:   runFileDownload,
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a file",
	Args:  cobra.ExactArgs(1),
	Run:   runFileDelete,
}

var fileHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage retained file history",
}

var fileHistoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List retained file transfers",
	Run:   runFileHistoryList,
}

var fileHistoryDownloadCmd = &cobra.Command{
	Use:   "download <history-id>",
	Short: "Download a retained file transfer",
	Long:  `Download a retained file transfer: arsenale file history download <history-id> --dest /tmp`,
	Args:  cobra.ExactArgs(1),
	Run:   runFileHistoryDownload,
}

var fileHistoryRestoreCmd = &cobra.Command{
	Use:   "restore <history-id>",
	Short: "Restore a retained file transfer",
	Long:  `Restore a retained file transfer: arsenale file history restore <history-id>`,
	Args:  cobra.ExactArgs(1),
	Run:   runFileHistoryRestore,
}

var fileHistoryDeleteCmd = &cobra.Command{
	Use:   "delete <history-id>",
	Short: "Delete a retained file transfer",
	Args:  cobra.ExactArgs(1),
	Run:   runFileHistoryDelete,
}

var (
	fileUploadPath     string
	fileDestDir        string
	fileConnection     string
	fileHistoryDestDir string
)

func init() {
	rootCmd.AddCommand(fileCmd)

	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileDownloadCmd)
	fileCmd.AddCommand(fileDeleteCmd)
	fileCmd.AddCommand(fileHistoryCmd)

	fileHistoryCmd.AddCommand(fileHistoryListCmd)
	fileHistoryCmd.AddCommand(fileHistoryDownloadCmd)
	fileHistoryCmd.AddCommand(fileHistoryRestoreCmd)
	fileHistoryCmd.AddCommand(fileHistoryDeleteCmd)

	fileUploadCmd.Flags().StringVar(&fileUploadPath, "file", "", "Path to the file to upload")
	fileUploadCmd.MarkFlagRequired("file")

	fileDownloadCmd.Flags().StringVar(&fileDestDir, "dest", ".", "Destination directory")
	fileListCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file transfer")
	fileUploadCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file transfer")
	fileDownloadCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file transfer")
	fileDeleteCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file transfer")
	fileHistoryListCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file history")
	fileHistoryDownloadCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file history")
	fileHistoryRestoreCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file history")
	fileHistoryDeleteCmd.Flags().StringVar(&fileConnection, "connection", "", "Connection name or ID for file history")
	fileHistoryDownloadCmd.Flags().StringVar(&fileHistoryDestDir, "dest", ".", "Destination directory")

	fileListCmd.MarkFlagRequired("connection")
	fileUploadCmd.MarkFlagRequired("connection")
	fileDownloadCmd.MarkFlagRequired("connection")
	fileDeleteCmd.MarkFlagRequired("connection")
	fileHistoryListCmd.MarkFlagRequired("connection")
	fileHistoryDownloadCmd.MarkFlagRequired("connection")
	fileHistoryRestoreCmd.MarkFlagRequired("connection")
	fileHistoryDeleteCmd.MarkFlagRequired("connection")
}

func runFileList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiGet("/api/files?connectionId="+url.QueryEscape(connectionID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, fileColumns)
}

func runFileUpload(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiUploadWithFields("/api/files", fileUploadPath, map[string]string{
		"connectionId": connectionID,
	}, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "name")
}

func runFileDownload(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	name := args[0]
	destPath := filepath.Join(fileDestDir, name)
	connectionID := resolveFileConnectionID(cfg)

	status, err := apiDownload("/api/files/"+url.PathEscape(name)+"?connectionId="+url.QueryEscape(connectionID), destPath, cfg)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal("download failed (HTTP %d)", status)
	}

	if !quiet {
		fmt.Printf("Downloaded %q to %s\n", name, destPath)
	}
}

func runFileDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiDelete("/api/files/"+url.PathEscape(args[0])+"?connectionId="+url.QueryEscape(connectionID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("File", args[0])
}

func runFileHistoryList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiGet("/api/files/history?connectionId="+url.QueryEscape(connectionID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if err := printer().Print(extractWrappedJSONField(body, "items"), fileHistoryColumns); err != nil {
		fatal("%v", err)
	}
}

func runFileHistoryDownload(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	historyID := args[0]
	connectionID := resolveFileConnectionID(cfg)
	historyEntry := resolveFileHistoryEntry(historyID, connectionID, cfg)
	destPath := resolveSSHDownloadDestination(historyEntry.FileName, strings.TrimSpace(fileHistoryDestDir))

	status, err := apiDownload("/api/files/history/"+url.PathEscape(historyID)+"?connectionId="+url.QueryEscape(connectionID), destPath, cfg)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal("download failed (HTTP %d)", status)
	}

	if !quiet {
		fmt.Printf("Downloaded history item %q to %s\n", historyEntry.FileName, destPath)
	}
}

func runFileHistoryRestore(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	historyID := args[0]
	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiPost("/api/files/history/"+url.PathEscape(historyID)+"/restore?connectionId="+url.QueryEscape(connectionID), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if err := printer().PrintSingle(extractWrappedJSONField(body, "item"), fileHistoryColumns); err != nil {
		fatal("%v", err)
	}
}

func runFileHistoryDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	connectionID := resolveFileConnectionID(cfg)
	body, status, err := apiDelete("/api/files/history/"+url.PathEscape(args[0])+"?connectionId="+url.QueryEscape(connectionID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("History item", args[0])
}

func resolveFileConnectionID(cfg *CLIConfig) string {
	conn, err := findConnectionByName(fileConnection, cfg)
	if err != nil {
		fatal("%v", err)
	}
	return conn.ID
}

type fileHistoryEntry struct {
	ID           string `json:"id"`
	FileName     string `json:"fileName"`
	RestoredName string `json:"restoredName"`
}

func resolveFileHistoryEntry(historyID, connectionID string, cfg *CLIConfig) fileHistoryEntry {
	body, status, err := apiGet("/api/files/history?connectionId="+url.QueryEscape(connectionID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	var payload struct {
		Items []fileHistoryEntry `json:"items"`
	}
	if err := json.Unmarshal(extractWrappedJSONField(body, "items"), &payload.Items); err != nil {
		fatal("parse history list: %v", err)
	}
	for _, item := range payload.Items {
		if item.ID == historyID {
			return item
		}
	}
	fatal("history item %q not found", historyID)
	return fileHistoryEntry{}
}
