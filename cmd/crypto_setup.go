package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"tkngate/internal/crypto"

	"github.com/pterm/pterm"
)

func ensureCryptoInitialized() error {
	spinner, _ := pterm.DefaultSpinner.Start("Initializing crypto engine...")
	err := crypto.InitCrypto()
	if err == nil {
		spinner.Success("Crypto Engine (AES-256) active")
		return nil
	}

	spinner.Warning("Crypto engine error: " + err.Error())

	options := []string{
		"Generate a new Master Key automatically",
		"Enter an existing Master Key manually",
		"Exit",
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.WithDefaultText("TKNGATE_MASTER_KEY is missing or invalid. How would you like to proceed?").WithOptions(options).Show()

	switch selectedOption {
	case "Generate a new Master Key automatically":
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return err
		}
		key := hex.EncodeToString(bytes)
		os.Setenv("TKNGATE_MASTER_KEY", key)

		boxContent := pterm.Sprintf("%s\n\n%s\n%s\n\nLinux/macOS: %s\nWindows:     %s",
			pterm.LightCyan(key),
			pterm.LightYellow("IMPORTANT: Copy this key and set it as an environment variable."),
			pterm.LightYellow("Do not lose this key! If lost, all donated mesh keys will be unrecoverable."),
			pterm.Gray("export TKNGATE_MASTER_KEY=\""+key+"\""),
			pterm.Gray("$env:TKNGATE_MASTER_KEY=\""+key+"\""))

		pterm.DefaultBox.WithTitle("New Master Key Generated").WithRightPadding(2).WithLeftPadding(2).Println(boxContent)

	case "Enter an existing Master Key manually":
		key, _ := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Enter your 32-character TKNGATE_MASTER_KEY")
		os.Setenv("TKNGATE_MASTER_KEY", key)

	default:
		os.Exit(1)
	}

	// Retry initialization recursively
	return ensureCryptoInitialized()
}
