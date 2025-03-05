package autopkg

import (
	"os"
	"strconv"
	"strings"
)

// Global Environment variables for GitHub Actions integration
var (
	DEBUG                  bool
	OVERRIDES_DIR          string
	RECIPE_TO_RUN          string
	TEAMS_WEBHOOK          string
	CLEANUP_LIST           string
	PROMOTE_LIST           string
	REPORT_PATH            string
	DISABLE_VERIFICATION   bool
	FORCE_UPDATE           bool
	FAIL_RECIPES           bool
	USE_BETA               bool
	AUTOPKG_REPO_LIST_PATH string

	// Uploader variables
	USE_JAMF_UPLOADER     bool
	USE_INTUNE_UPLOADER   bool
	JAMFPRO_URL           string
	JAMFPRO_CLIENT_ID     string
	JAMFPRO_CLIENT_SECRET string
	INTUNE_TENANT_ID      string
	INTUNE_CLIENT_ID      string
	INTUNE_CLIENT_SECRET  string
	//AUTOPKG_REPOS         []string
	RECIPE_LISTS      []string
	PRIVATE_REPO_URL  string
	PRIVATE_REPO_PATH string
	JCDS2_MODE        bool
	API_USERNAME      string
	API_PASSWORD      string
	SMB_URL           string
	SMB_USERNAME      string
	SMB_PASSWORD      string
)

// LoadEnvironmentVariables loads all environment variables used by the package
func LoadEnvironmentVariables() {
	// Check if DEBUG is enabled
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" {
		DEBUG, _ = strconv.ParseBool(debugEnv)
	}

	// Get overrides directory
	OVERRIDES_DIR = os.Getenv("OVERRIDES_DIR")

	// Get autopkg repo list
	AUTOPKG_REPO_LIST_PATH = os.Getenv("AUTOPKG_REPO_LIST_PATH")
	// Get recipe to run
	RECIPE_TO_RUN = os.Getenv("RECIPE")

	// Get Teams webhook URL
	TEAMS_WEBHOOK = os.Getenv("TEAMS_WEBHOOK")

	// Get cleanup list path
	CLEANUP_LIST = os.Getenv("CLEANUP_LIST")

	// Get promote list path
	PROMOTE_LIST = os.Getenv("PROMOTE_LIST")

	// Get verification disable flag
	disableVerificationEnv := os.Getenv("DISABLE_VERIFICATION")
	if disableVerificationEnv != "" {
		DISABLE_VERIFICATION, _ = strconv.ParseBool(disableVerificationEnv)
	}

	// Get report path
	REPORT_PATH = os.Getenv("REPORT_PATH")
	if REPORT_PATH == "" {
		REPORT_PATH = "/tmp/autopkg.plist"
	}

	// Check if beta version should be used
	useBetaStr := os.Getenv("USE_BETA")
	if useBetaStr != "" {
		USE_BETA, _ = strconv.ParseBool(useBetaStr)
	}

	// GitHub Actions specific variables
	useJamfStr := os.Getenv("USE_JAMF_UPLOADER")
	if useJamfStr != "" {
		USE_JAMF_UPLOADER, _ = strconv.ParseBool(useJamfStr)
	}

	useIntuneStr := os.Getenv("USE_INTUNE_UPLOADER")
	if useIntuneStr != "" {
		USE_INTUNE_UPLOADER, _ = strconv.ParseBool(useIntuneStr)
	}

	// Jamf Pro settings
	JAMFPRO_URL = os.Getenv("JAMFPRO_URL")
	API_USERNAME = os.Getenv("API_USERNAME")
	API_PASSWORD = os.Getenv("API_PASSWORD")
	JAMFPRO_CLIENT_ID = os.Getenv("JAMFPRO_CLIENT_ID")
	JAMFPRO_CLIENT_SECRET = os.Getenv("JAMFPRO_CLIENT_SECRET")
	SMB_URL = os.Getenv("SMB_URL")
	SMB_USERNAME = os.Getenv("SMB_USERNAME")
	SMB_PASSWORD = os.Getenv("SMB_PASSWORD")

	jcds2ModeStr := os.Getenv("JCDS2_MODE")
	if jcds2ModeStr != "" {
		JCDS2_MODE, _ = strconv.ParseBool(jcds2ModeStr)
	}

	// Intune settings
	INTUNE_TENANT_ID = os.Getenv("INTUNE_TENANT_ID")
	INTUNE_CLIENT_ID = os.Getenv("INTUNE_CLIENT_ID")
	INTUNE_CLIENT_SECRET = os.Getenv("INTUNE_CLIENT_SECRET")

	// Recipe lists
	listsStr := os.Getenv("RECIPE_LISTS")
	if listsStr != "" {
		for _, list := range strings.Split(listsStr, ",") {
			list = strings.TrimSpace(list)
			if list != "" {
				RECIPE_LISTS = append(RECIPE_LISTS, list)
			}
		}
	}

	// Private repo settings
	PRIVATE_REPO_URL = os.Getenv("PRIVATE_REPO_URL")
	PRIVATE_REPO_PATH = os.Getenv("PRIVATE_REPO_PATH")

	// General settings
	forceUpdateStr := os.Getenv("FORCE_UPDATE")
	if forceUpdateStr != "" {
		FORCE_UPDATE, _ = strconv.ParseBool(forceUpdateStr)
	}

	failRecipesStr := os.Getenv("FAIL_RECIPES")
	if failRecipesStr != "" {
		FAIL_RECIPES, _ = strconv.ParseBool(failRecipesStr)
	}
}
