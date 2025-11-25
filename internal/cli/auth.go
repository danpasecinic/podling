package cli

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(whoamiCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login [username]",
	Short: "Login to the Podling master",
	Long:  `Authenticates with the Podling master and displays the JWT token.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())

		var username string
		if len(args) > 0 {
			username = args[0]
		} else {
			fmt.Print("Username: ")
			_, _ = fmt.Scanln(&username)
		}

		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		resp, err := client.Login(username, string(passwordBytes))
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Printf("Login successful!\n")
		fmt.Printf("User: %s (role: %s)\n", resp.User.Username, resp.User.Role)
		fmt.Printf("Token expires: %s\n\n", resp.ExpiresAt)

		fmt.Println("To use this token, set the environment variable:")
		fmt.Printf("  export PODLING_TOKEN=%s\n\n", resp.Token)
		fmt.Println("Or use the --token flag:")
		fmt.Printf("  podling --token=%s <command>\n", resp.Token)

		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user",
	Long:  `Displays information about the currently authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewAuthenticatedClient()

		user, err := client.GetCurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		fmt.Printf("User ID:   %v\n", user["userId"])
		fmt.Printf("Username:  %v\n", user["username"])
		fmt.Printf("Role:      %v\n", user["role"])
		fmt.Printf("Auth Type: %v\n", user["authType"])

		if nodeID, ok := user["nodeId"]; ok && nodeID != "" {
			fmt.Printf("Node ID:   %v\n", nodeID)
		}

		return nil
	},
}
