package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/zchee/abm"
	"golang.org/x/oauth2"
)

// NewRequestCmd creates the request command.
func NewRequestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "request <url>",
		Short: "Make an authenticated ABM API request",
		Long:  "Makes an authenticated GET request to the given Apple Business Manager API URL and prints the response body.",
		Args:  cobra.ExactArgs(1),
		RunE:  runRequest,
	}
}

func runRequest(cmd *cobra.Command, args []string) error {
	if err := Cfg.ValidateABM(); err != nil {
		return err
	}

	ctx := context.Background()

	assertion, err := abm.NewAssertion(ctx, Cfg.ABM.ClientID, Cfg.ABM.KeyID, Cfg.ABM.PrivateKey)
	if err != nil {
		return fmt.Errorf("creating ABM assertion: %w", err)
	}

	ts, err := abm.NewTokenSource(ctx, nil, Cfg.ABM.ClientID, assertion, "")
	if err != nil {
		return fmt.Errorf("creating ABM token source: %w", err)
	}

	client := oauth2.NewClient(ctx, ts)

	resp, err := client.Get(args[0])
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Fprintf(cmd.ErrOrStderr(), "HTTP %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	for k, v := range resp.Header {
		for _, val := range v {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", k, val)
		}
	}
	fmt.Fprintln(cmd.ErrOrStderr())

	_, err = io.Copy(cmd.OutOrStdout(), resp.Body)
	return err
}
