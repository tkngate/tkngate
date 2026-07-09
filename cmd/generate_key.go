package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var generateKeyCmd = &cobra.Command{
	Use:   "generate-master-key",
	Short: "Generates a new secure 32-byte TKNGATE_MASTER_KEY and saves it to .env",
	Run: func(cmd *cobra.Command, args []string) {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			pterm.Error.Println("Failed to generate secure random bytes:", err)
			return
		}
		key := hex.EncodeToString(bytes)

		boxContent := pterm.Sprintf("%s\n\n%s\n%s\n\nLinux/macOS: %s\nWindows:     %s",
			pterm.LightCyan(key),
			Gold("IMPORTANT: Copy this key and set it as an environment variable."),
			Gold("Do not lose this key! If lost, all donated mesh keys will be unrecoverable."),
			Parch("export TKNGATE_MASTER_KEY=\""+key+"\""),
			Parch("$env:TKNGATE_MASTER_KEY=\""+key+"\""))

		pterm.DefaultBox.WithTitle("New Master Key Generated").WithRightPadding(2).WithLeftPadding(2).Println(boxContent)

		// Automatically save to .env
		saveKeyToEnv(key)
	},
}

// saveKeyToEnv writes or updates the TKNGATE_MASTER_KEY in the .env file.
func saveKeyToEnv(key string) {
	envFile := ".env"
	content, err := os.ReadFile(envFile)
	var newContent []string
	keyUpdated := false

	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "TKNGATE_MASTER_KEY=") {
				newContent = append(newContent, "TKNGATE_MASTER_KEY="+key)
				keyUpdated = true
			} else {
				newContent = append(newContent, line)
			}
		}
	}

	if !keyUpdated {
		newContent = append(newContent, "TKNGATE_MASTER_KEY="+key)
	}

	err = os.WriteFile(envFile, []byte(strings.Join(newContent, "\n")), 0644)
	if err != nil {
		pterm.Error.Println("Failed to update .env file:", err)
	} else {
		pterm.Success.Printf("Automatically saved Master Key to %s\n", envFile)
	}
}

func init() {
	rootCmd.AddCommand(generateKeyCmd)
}
