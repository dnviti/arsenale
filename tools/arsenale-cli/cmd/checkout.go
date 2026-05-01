package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var checkoutColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "CONNECTION", Field: "connection"},
	{Header: "USER", Field: "user"},
	{Header: "STATUS", Field: "status"},
	{Header: "CREATED_AT", Field: "createdAt"},
}

// ---------------------------------------------------------------------------
// Top-level: arsenale checkout
// ---------------------------------------------------------------------------

var checkoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Manage connection checkouts",
}

var checkoutListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all checkouts",
	Run:   runCheckoutList,
}

var checkoutGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get checkout details",
	Args:  cobra.ExactArgs(1),
	Run:   runCheckoutGet,
}

var checkoutCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new checkout",
	Long: `Create a checkout from a JSON/YAML file or with flags:
  arsenale checkout create --from-file checkout.yaml
  arsenale checkout create --connection-id <connId> --reason "maintenance"`,
	Run: runCheckoutCreate,
}

var checkoutApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a checkout request",
	Args:  cobra.ExactArgs(1),
	Run:   runCheckoutApprove,
}

var checkoutRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a checkout request",
	Args:  cobra.ExactArgs(1),
	Run:   runCheckoutReject,
}

var checkoutCheckinCmd = &cobra.Command{
	Use:   "checkin <id>",
	Short: "Check in a checked-out connection",
	Args:  cobra.ExactArgs(1),
	Run:   runCheckoutCheckin,
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

var (
	checkoutFromFile     string
	checkoutConnectionID string
	checkoutReason       string
)

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(checkoutCmd)

	checkoutCmd.AddCommand(checkoutListCmd)
	checkoutCmd.AddCommand(checkoutGetCmd)
	checkoutCmd.AddCommand(checkoutCreateCmd)
	checkoutCmd.AddCommand(checkoutApproveCmd)
	checkoutCmd.AddCommand(checkoutRejectCmd)
	checkoutCmd.AddCommand(checkoutCheckinCmd)

	checkoutCreateCmd.Flags().StringVarP(&checkoutFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	checkoutCreateCmd.Flags().StringVar(&checkoutConnectionID, "connection-id", "", "Connection ID to check out")
	checkoutCreateCmd.Flags().StringVar(&checkoutReason, "reason", "", "Reason for checkout")

	checkoutRejectCmd.Flags().StringVar(&checkoutReason, "reason", "", "Reason for rejection")
}

// ===========================================================================
// Run functions
// ===========================================================================

func runCheckoutList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/checkouts", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, checkoutColumns)
}

func runCheckoutGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/checkouts/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, checkoutColumns)
}

func runCheckoutCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if checkoutFromFile != "" {
		data, err = readResourceFromFileOrStdin(checkoutFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if checkoutConnectionID == "" {
			fatal("provide --from-file or --connection-id")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"connectionId": checkoutConnectionID,
			"reason":       checkoutReason,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost("/api/checkouts", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runCheckoutApprove(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/checkouts/%s/approve", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Checkout %q approved\n", args[0])
	}
}

func runCheckoutReject(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var payload interface{}
	if checkoutReason != "" {
		payload = map[string]string{"reason": checkoutReason}
	}

	body, status, err := apiPost(fmt.Sprintf("/api/checkouts/%s/reject", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Checkout %q rejected\n", args[0])
	}
}

func runCheckoutCheckin(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/checkouts/%s/checkin", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Checkout %q checked in\n", args[0])
	}
}
