package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var fileColumns = []Column{
	{Header: "NAME", Field: "name"},
	{Header: "SIZE", Field: "size"},
	{Header: "CREATED_AT", Field: "createdAt"},
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage files",
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

var (
	fileUploadPath string
	fileDestDir    string
)

func init() {
	rootCmd.AddCommand(fileCmd)

	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileDownloadCmd)
	fileCmd.AddCommand(fileDeleteCmd)

	fileUploadCmd.Flags().StringVar(&fileUploadPath, "file", "", "Path to the file to upload")
	fileUploadCmd.MarkFlagRequired("file")

	fileDownloadCmd.Flags().StringVar(&fileDestDir, "dest", ".", "Destination directory")
}

func runFileList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/files", cfg)
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

	body, status, err := apiUpload("/api/files", fileUploadPath, cfg)
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

	status, err := apiDownload("/api/files/"+name, destPath, cfg)
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

	body, status, err := apiDelete("/api/files/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("File", args[0])
}
