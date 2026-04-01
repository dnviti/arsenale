package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var recordingColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "SESSION_ID", Field: "sessionId"},
	{Header: "USER", Field: "user"},
	{Header: "TYPE", Field: "type"},
	{Header: "DURATION", Field: "duration"},
	{Header: "CREATED_AT", Field: "createdAt"},
}

var recordingCmd = &cobra.Command{
	Use:   "recording",
	Short: "Manage session recordings",
}

var recordingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recordings",
	Run:   runRecordingList,
}

var recordingGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get recording details",
	Args:  cobra.ExactArgs(1),
	Run:   runRecordingGet,
}

var recordingDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a recording",
	Args:  cobra.ExactArgs(1),
	Run:   runRecordingDelete,
}

var recordingAuditTrailCmd = &cobra.Command{
	Use:   "audit-trail <id>",
	Short: "Get recording audit trail",
	Args:  cobra.ExactArgs(1),
	Run:   runRecordingAuditTrail,
}

var recordingAnalyzeCmd = &cobra.Command{
	Use:   "analyze <id>",
	Short: "Analyze a recording",
	Args:  cobra.ExactArgs(1),
	Run:   runRecordingAnalyze,
}

var recordingExportVideoCmd = &cobra.Command{
	Use:   "export-video <id>",
	Short: "Export recording as video file",
	Args:  cobra.ExactArgs(1),
	Run:   runRecordingExportVideo,
}

var recordingExportDest string

func init() {
	rootCmd.AddCommand(recordingCmd)

	recordingCmd.AddCommand(recordingListCmd)
	recordingCmd.AddCommand(recordingGetCmd)
	recordingCmd.AddCommand(recordingDeleteCmd)
	recordingCmd.AddCommand(recordingAuditTrailCmd)
	recordingCmd.AddCommand(recordingAnalyzeCmd)
	recordingCmd.AddCommand(recordingExportVideoCmd)

	recordingExportVideoCmd.Flags().StringVar(&recordingExportDest, "dest", "", "Destination file path (default: recording-<id>.mp4)")
}

func runRecordingList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/recordings", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, recordingColumns)
}

func runRecordingGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/recordings/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, recordingColumns)
}

func runRecordingDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/recordings/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Recording", args[0])
}

func runRecordingAuditTrail(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/recordings/%s/audit-trail", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "TIMESTAMP", Field: "timestamp"},
		{Header: "ACTION", Field: "action"},
		{Header: "USER", Field: "user"},
	})
}

func runRecordingAnalyze(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/recordings/%s/analyze", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "STATUS", Field: "status"},
		{Header: "SUMMARY", Field: "summary"},
	})
}

func runRecordingExportVideo(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	dest := recordingExportDest
	if dest == "" {
		dest = fmt.Sprintf("recording-%s.mp4", args[0])
	}

	status, err := apiDownload(fmt.Sprintf("/api/recordings/%s/video", args[0]), dest, cfg)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal("download failed (HTTP %d)", status)
	}

	if !quiet {
		fmt.Printf("Recording exported to %s\n", dest)
	}
}
