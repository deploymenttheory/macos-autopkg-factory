package autopkg

import (
	"fmt"
	"os"
	"os/exec"
)

// ConfigureJamfUploader sets up JamfUploader settings
func ConfigureJamfUploader(config *Config, prefsPath string) error {
	// Check for JSS_URL from config or environment
	jssURL := config.JAMFPRO_URL
	if jssURL == "" {
		jssURL = os.Getenv("JAMFPRO_URL")
	}

	// Only proceed if JSS_URL is set
	if jssURL == "" {
		return nil
	}

	// Set JSS_URL
	cmd := exec.Command("defaults", "write", prefsPath, "JSS_URL", jssURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set JSS_URL: %w", err)
	}

	// Set API_USERNAME if provided in config or environment
	apiUsername := config.API_USERNAME
	if apiUsername == "" {
		apiUsername = os.Getenv("API_USERNAME")
	}
	if apiUsername != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "API_USERNAME", apiUsername)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set API_USERNAME: %w", err)
		}
	}

	// Set API_PASSWORD if provided in config or environment
	apiPassword := config.API_PASSWORD
	if apiPassword == "" {
		apiPassword = os.Getenv("API_PASSWORD")
	}
	if apiPassword != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "API_PASSWORD", apiPassword)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set API_PASSWORD: %w", err)
		}
	}

	// Set CLIENT_ID if provided in config or environment
	clientID := config.JAMFPRO_CLIENT_ID
	if clientID == "" {
		clientID = os.Getenv("CLIENT_ID")
	}
	if clientID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_ID", clientID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_ID: %w", err)
		}
	}

	// Set CLIENT_SECRET if provided in config or environment
	clientSecret := config.JAMFPRO_CLIENT_SECRET
	if clientSecret == "" {
		clientSecret = os.Getenv("CLIENT_SECRET")
	}
	if clientSecret != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_SECRET", clientSecret)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_SECRET: %w", err)
		}
	}

	// Set SMB_URL if provided in config or environment
	smbURL := config.SMB_URL
	if smbURL == "" {
		smbURL = os.Getenv("SMB_URL")
	}
	if smbURL != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_URL", smbURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_URL: %w", err)
		}
	}

	// Set SMB_USERNAME if provided in config or environment
	smbUsername := config.SMB_USERNAME
	if smbUsername == "" {
		smbUsername = os.Getenv("SMB_USERNAME")
	}
	if smbUsername != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_USERNAME", smbUsername)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_USERNAME: %w", err)
		}
	}

	// Set SMB_PASSWORD if provided in config or environment
	smbPassword := config.SMB_PASSWORD
	if smbPassword == "" {
		smbPassword = os.Getenv("SMB_PASSWORD")
	}
	if smbPassword != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_PASSWORD", smbPassword)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_PASSWORD: %w", err)
		}
	}

	Logger("JamfUploader configured.", LogSuccess)
	return nil
}

// ConfigureIntuneUploader configures the Microsoft Intune integration settings
func ConfigureIntuneUploader(config *Config, prefsPath string) error {
	// Set CLIENT_ID if provided in config or environment
	clientID := config.INTUNE_CLIENT_ID
	if clientID == "" {
		clientID = os.Getenv("INTUNE_CLIENT_ID")
	}
	if clientID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_ID", clientID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_ID: %w", err)
		}
		Logger(fmt.Sprintf("Set (Intune) CLIENT_ID in %s", prefsPath), LogSuccess)
	}

	// Set CLIENT_SECRET if provided in config or environment
	clientSecret := config.INTUNE_CLIENT_SECRET
	if clientSecret == "" {
		clientSecret = os.Getenv("INTUNE_CLIENT_SECRET")
	}
	if clientSecret != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_SECRET", clientSecret)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_SECRET: %w", err)
		}
		Logger(fmt.Sprintf("Set (Intune) CLIENT_SECRET in %s", prefsPath), LogSuccess)
	}

	// Set TENANT_ID if provided in config or environment
	tenantID := config.INTUNE_TENANT_ID
	if tenantID == "" {
		tenantID = os.Getenv("INTUNE_TENANT_ID")
	}
	if tenantID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "TENANT_ID", tenantID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set TENANT_ID: %w", err)
		}
		Logger(fmt.Sprintf("Set (Intune) TENANT_ID in %s", prefsPath), LogSuccess)
	}

	// Check if we set at least some of the Intune configuration
	if clientID != "" || clientSecret != "" || tenantID != "" {
		Logger("IntuneUploader configured.", LogSuccess)
	}

	return nil
}
