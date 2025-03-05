package autopkg

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"howett.net/plist"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// Recipe represents an AutoPkg recipe
type Recipe struct {
	Path     string
	Error    bool
	Results  map[string]interface{}
	Updated  bool
	Removed  bool
	Promoted bool
	Verified *bool

	plist  map[string]interface{}
	hasRun bool
}

// NewRecipe creates a new Recipe instance
func NewRecipe(path string, overridesDir string) *Recipe {
	fullPath := path
	if overridesDir != "" {
		fullPath = filepath.Join(overridesDir, path)
	}

	return &Recipe{
		Path:     fullPath,
		Error:    false,
		Results:  make(map[string]interface{}),
		Updated:  false,
		Removed:  false,
		Promoted: false,
		Verified: nil,
		hasRun:   false,
	}
}

// LoadPlist loads the recipe's plist data
func (r *Recipe) LoadPlist() error {
	if r.plist != nil {
		return nil
	}

	// Read plist file
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return fmt.Errorf("failed to read recipe file: %w", err)
	}

	// Use howett.net/plist to unmarshal the data
	r.plist = make(map[string]interface{})

	// Decode the plist data
	_, err = plist.Unmarshal(data, &r.plist)
	if err != nil {
		return fmt.Errorf("failed to parse plist: %w", err)
	}

	return nil
}

// Name returns the recipe name
func (r *Recipe) Name() (string, error) {
	if err := r.LoadPlist(); err != nil {
		return "", err
	}

	if input, ok := r.plist["Input"].(map[string]interface{}); ok {
		if name, ok := input["NAME"].(string); ok {
			return name, nil
		}
	}

	return "Recipe", nil
}

// Identifier returns the recipe identifier
func (r *Recipe) Identifier() (string, error) {
	if err := r.LoadPlist(); err != nil {
		return "", err
	}

	if identifier, ok := r.plist["Identifier"].(string); ok {
		return identifier, nil
	}

	return "", nil
}

// UpdatedVersion returns the updated version after a run
func (r *Recipe) UpdatedVersion() string {
	if !r.hasRun || r.Results == nil {
		return ""
	}

	if imported, ok := r.Results["imported"].([]interface{}); ok && len(imported) > 0 {
		if item, ok := imported[0].(map[string]interface{}); ok {
			if version, ok := item["version"].(string); ok {
				return strings.TrimSpace(strings.ReplaceAll(version, " ", ""))
			}
		}
	}

	return ""
}

// VerifyTrustInfo verifies trust info for the recipe
func (r *Recipe) VerifyTrustInfo(debug bool) (bool, error) {
	identifier, err := r.Identifier()
	if err != nil {
		return false, err
	}

	name, _ := r.Name()
	logger.Logger(fmt.Sprintf("Verifying trust info for recipe: %s", name), logger.LogInfo)

	cmdArgs := []string{
		"verify-trust-info",
		fmt.Sprintf("\"%s\"", identifier),
		"-vvv",
	}

	cmd := exec.Command("/usr/local/bin/autopkg", cmdArgs...)

	if debug {
		logger.Logger(fmt.Sprintf("Running: autopkg %s", strings.Join(cmdArgs, " ")), logger.LogDebug)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		r.Results["message"] = string(output)
		verified := false
		r.Verified = &verified
		logger.Logger(fmt.Sprintf("Trust verification failed for %s: %v", name, err), logger.LogError)
		return false, fmt.Errorf("trust verification failed: %w", err)
	}

	verified := true
	r.Verified = &verified
	logger.Logger(fmt.Sprintf("Trust verification succeeded for %s", name), logger.LogSuccess)
	return true, nil
}

// UpdateTrustInfo updates trust info for the recipe
func (r *Recipe) UpdateTrustInfo(debug bool) error {
	identifier, err := r.Identifier()
	if err != nil {
		return err
	}

	name, _ := r.Name()
	logger.Logger(fmt.Sprintf("Updating trust info for recipe: %s", name), logger.LogInfo)

	cmdArgs := []string{
		"update-trust-info",
		fmt.Sprintf("\"%s\"", identifier),
	}

	cmd := exec.Command("/usr/local/bin/autopkg", cmdArgs...)

	if debug {
		logger.Logger(fmt.Sprintf("Running: autopkg %s", strings.Join(cmdArgs, " ")), logger.LogDebug)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Logger(fmt.Sprintf("Failed to update trust info for %s: %v", name, err), logger.LogError)
		return fmt.Errorf("failed to update trust info: %s, %w", output, err)
	}

	logger.Logger(fmt.Sprintf("Successfully updated trust info for %s", name), logger.LogSuccess)
	return nil
}

// ParseReport parses an AutoPkg report plist
func ParseReport(reportPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	// Parse plist using howett.net/plist
	var reportData map[string]interface{}
	_, err = plist.Unmarshal(data, &reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse report plist: %w", err)
	}

	// Process the report data
	results := map[string]interface{}{
		"imported": []interface{}{},
		"failed":   []interface{}{},
		"removed":  []interface{}{},
		"promoted": []interface{}{},
	}

	// Extract failures
	if failures, ok := reportData["failures"].([]interface{}); ok {
		results["failed"] = failures
	}

	// Extract items from summary_results
	if summaryResults, ok := reportData["summary_results"].(map[string]interface{}); ok {
		// Extract imported items
		if intuneResults, ok := summaryResults["intuneappuploader_summary_result"].(map[string]interface{}); ok {
			if dataRows, ok := intuneResults["data_rows"].([]interface{}); ok {
				results["imported"] = dataRows
			}
		}

		// Extract removed items
		if removedResults, ok := summaryResults["intuneappcleaner_summary_result"].(map[string]interface{}); ok {
			if dataRows, ok := removedResults["data_rows"].([]interface{}); ok {
				results["removed"] = dataRows
			}
		}

		// Extract promoted items
		if promotedResults, ok := summaryResults["intuneapppromoter_summary_result"].(map[string]interface{}); ok {
			if dataRows, ok := promotedResults["data_rows"].([]interface{}); ok {
				results["promoted"] = dataRows
			}
		}
	}

	return results, nil
}

// ParseList parses a list of apps from a JSON file
func ParseList(listPath string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read list file: %w", err)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to parse JSON list: %w", err)
	}

	return list, nil
}

// RecipeOptions holds options for running recipes
type RecipeOptions struct {
	DisableVerification bool
	CleanupList         string
	PromoteList         string
	Debug               bool
}

// Run runs an AutoPkg recipe
func (r *Recipe) Run(opts *RecipeOptions) (map[string]interface{}, error) {
	// Skip if verification failed and is required
	if !opts.DisableVerification && r.Verified != nil && !*r.Verified {
		r.Error = true
		r.Results["failed"] = true
		return r.Results, fmt.Errorf("recipe verification failed")
	}

	// Create report file if it doesn't exist
	reportPath := "/tmp/autopkg.plist"
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		file, err := os.Create(reportPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create report file: %w", err)
		}
		file.Close()
	}

	// Get recipe identifier
	identifier, err := r.Identifier()
	if err != nil {
		return nil, err
	}

	// Build command
	verbosityLevel := "-vvv"
	if !opts.Debug {
		verbosityLevel = "-v"
	}

	cmdArgs := []string{
		"run",
		fmt.Sprintf("\"%s\"", identifier),
		verbosityLevel,
		"--report-plist",
		reportPath,
	}

	// Add cleanup options if specified
	if opts.CleanupList != "" {
		cleanupApps, err := ParseList(opts.CleanupList)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cleanup list: %w", err)
		}

		name, _ := r.Name()
		foundApp := false
		for _, app := range cleanupApps {
			if appName, ok := app["name"].(string); ok && appName == name {
				cmdArgs = append(cmdArgs, "--post", "com.github.almenscorner.intune-upload.processors/IntuneAppCleaner")
				logger.Logger(fmt.Sprintf("Adding cleanup processor for %s", name), logger.LogInfo)

				// Add keep count if specified
				if keepCount, ok := app["keep_count"].(float64); ok {
					cmdArgs = append(cmdArgs, "-k", fmt.Sprintf("keep_version_count=%d", int(keepCount)))
					logger.Logger(fmt.Sprintf("Setting keep count to %d for %s", int(keepCount), name), logger.LogInfo)
				}
				foundApp = true
				break
			}
		}

		if !foundApp && opts.Debug {
			logger.Logger(fmt.Sprintf("Skipping cleanup for %s, not in cleanup list", name), logger.LogWarning)
		}
	}

	// Add promotion options if specified
	if opts.PromoteList != "" {
		promoteApps, err := ParseList(opts.PromoteList)
		if err != nil {
			return nil, fmt.Errorf("failed to parse promote list: %w", err)
		}

		name, _ := r.Name()
		foundApp := false
		for _, app := range promoteApps {
			if appName, ok := app["name"].(string); ok && appName == name {
				cmdArgs = append(cmdArgs, "--post", "com.github.almenscorner.intune-upload.processors/IntuneAppPromoter")
				logger.Logger(fmt.Sprintf("Adding promotion processor for %s", name), logger.LogInfo)
				foundApp = true
				break
			}
		}

		if !foundApp && opts.Debug {
			logger.Logger(fmt.Sprintf("Skipping promotion for %s, not in promote list", name), logger.LogWarning)
		}
	}

	// Prepare command
	cmd := exec.Command("/usr/local/bin/autopkg", cmdArgs...)

	if opts.Debug {
		logger.Logger(fmt.Sprintf("Running: autopkg %s", strings.Join(cmdArgs, " ")), logger.LogDebug)
	}

	// Set up pipes to capture and display output in real-time
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Process stdout output in real-time
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Process stderr output in real-time
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				os.Stderr.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		r.Error = true
	}

	r.hasRun = true

	// Parse the report
	results, err := ParseReport(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	r.Results = results

	// Set flags based on results
	if _, failed := r.Results["failed"]; !failed && !r.Error && r.UpdatedVersion() != "" {
		r.Updated = true
	}

	if removed, ok := r.Results["removed"].([]interface{}); ok && len(removed) > 0 {
		if item, ok := removed[0].(map[string]interface{}); ok {
			if count, ok := item["removed count"].(string); ok && count != "0" {
				r.Removed = true
			}
		}
	}

	if promoted, ok := r.Results["promoted"].([]interface{}); ok && len(promoted) > 0 {
		if item, ok := promoted[0].(map[string]interface{}); ok {
			if _, ok := item["promotions"]; ok {
				r.Promoted = true
			}
		}
	}

	return r.Results, nil
}

// ParseRecipes parses a recipe list file or individual recipe paths
func ParseRecipes(recipesPath string, overridesDir string) ([]*Recipe, error) {
	var recipes []*Recipe

	// Check if RECIPE_TO_RUN is set, process as comma-separated list
	if RECIPE_TO_RUN != "" {
		recipeNames := strings.Split(RECIPE_TO_RUN, ", ")
		for _, name := range recipeNames {
			name = strings.TrimSpace(name)
			// Ensure recipe has .recipe extension
			if !strings.HasSuffix(name, ".recipe") {
				name += ".recipe"
			}
			recipes = append(recipes, NewRecipe(name, overridesDir))
		}
		return recipes, nil
	}

	// Check if it's a JSON or plist file listing multiple recipes
	ext := filepath.Ext(recipesPath)
	if ext == ".json" || ext == ".plist" {
		// Parse file as a list
		var recipeNames []string

		// Read file
		data, err := os.ReadFile(recipesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read recipe list file: %w", err)
		}

		// Parse based on file type
		if ext == ".json" {
			if err := json.Unmarshal(data, &recipeNames); err != nil {
				return nil, fmt.Errorf("failed to parse JSON recipe list: %w", err)
			}
		} else {
			// For plist, we'd need a proper plist parser
			// This is simplified and would need enhancement
			return nil, fmt.Errorf("plist recipe lists not yet supported")
		}

		// Create Recipe objects for each name
		for _, name := range recipeNames {
			recipes = append(recipes, NewRecipe(name, overridesDir))
		}
	} else {
		// Assume it's a comma-separated list of recipe names
		recipeNames := strings.Split(recipesPath, ",")
		for _, name := range recipeNames {
			name = strings.TrimSpace(name)
			// Ensure recipe has .recipe extension
			if !strings.HasSuffix(name, ".recipe") {
				name += ".recipe"
			}
			recipes = append(recipes, NewRecipe(name, overridesDir))
		}
	}

	return recipes, nil
}

// HandleRecipe processes a recipe with verification and execution
func HandleRecipe(recipe *Recipe, opts *RecipeOptions) error {
	// Verify trust info if not disabled
	if !opts.DisableVerification {
		verified, err := recipe.VerifyTrustInfo(opts.Debug)
		if err != nil && !verified {
			if err := recipe.UpdateTrustInfo(opts.Debug); err != nil {
				return fmt.Errorf("failed to update trust info: %w", err)
			}
		}
	}

	// Run the recipe
	if recipe.Verified == nil || *recipe.Verified {
		if _, err := recipe.Run(opts); err != nil {
			return fmt.Errorf("recipe run failed: %w", err)
		}
	}

	return nil
}

// ProcessRecipes handles multiple recipes with verification and notification support
func ProcessRecipes(recipesPath string, overridesDir string, opts *RecipeOptions, teamsWebhook string) error {
	// Load environment variables if not already set
	if OVERRIDES_DIR == "" {
		LoadEnvironmentVariables()
	}

	// Use environment variable for overrides directory if not provided
	if overridesDir == "" {
		overridesDir = OVERRIDES_DIR
	}

	// Parse recipes
	recipes, err := ParseRecipes(recipesPath, overridesDir)
	if err != nil {
		Logger(fmt.Sprintf("Failed to parse recipes: %v", err), LogError)
		return fmt.Errorf("failed to parse recipes: %w", err)
	}

	Logger(fmt.Sprintf("Processing %d recipes", len(recipes)), LogInfo)

	var failures []*Recipe
	var successes []*Recipe
	var updates []*Recipe

	// Process each recipe
	for _, recipe := range recipes {
		name, _ := recipe.Name()
		logger.Logger(fmt.Sprintf("Processing recipe: %s", name), logger.LogInfo)

		if err := HandleRecipe(recipe, opts); err != nil {
			logger.Logger(fmt.Sprintf("Error handling recipe %s: %v", recipe.Path, err), logger.LogError)
			failures = append(failures, recipe)
		} else if recipe.Updated {
			logger.Logger(fmt.Sprintf("Recipe %s updated to version %s", name, recipe.UpdatedVersion()), logger.LogSuccess)
			updates = append(updates, recipe)
			successes = append(successes, recipe)
		} else if recipe.Removed {
			logger.Logger(fmt.Sprintf("Recipe %s cleaned up old versions", name), logger.LogSuccess)
			successes = append(successes, recipe)
		} else if recipe.Promoted {
			logger.Logger(fmt.Sprintf("Recipe %s promoted to production", name), logger.LogSuccess)
			successes = append(successes, recipe)
		} else {
			logger.Logger(fmt.Sprintf("Recipe %s processed with no changes", name), logger.LogInfo)
			successes = append(successes, recipe)
		}

		// Send Teams notification if configured
		webhookToUse := teamsWebhook
		if webhookToUse == "" {
			webhookToUse = TEAMS_WEBHOOK
		}

		if webhookToUse != "" && !opts.Debug {
			if err := NotifyTeams(recipe, webhookToUse); err != nil {
				logger.Logger(fmt.Sprintf("Error sending Teams notification: %v", err), logger.LogWarning)
			} else {
				logger.Logger("Teams notification sent successfully", logger.LogInfo)
			}
		} else if opts.Debug {
			logger.Logger("Skipping Teams notification - debug is enabled", logger.LogDebug)
		} else if webhookToUse == "" {
			logger.Logger("Skipping Teams notification - webhook URL is missing", logger.LogWarning)
		}

		// Track verification failures for reporting
		if !opts.DisableVerification && recipe.Verified != nil && !*recipe.Verified {
			failures = append(failures, recipe)
		}
	}

	// Report on final status
	logger.Logger(fmt.Sprintf("Recipe processing completed: %d total, %d successes, %d failures, %d updates",
		len(recipes), len(successes), len(failures), len(updates)), logger.LogInfo)

	// Report on failures
	if len(failures) > 0 {
		logger.Logger(fmt.Sprintf("Verification failed for %d recipes:", len(failures)), logger.LogError)
		for _, recipe := range failures {
			name, _ := recipe.Name()
			logger.Logger(fmt.Sprintf("  - %s (%s)", name, recipe.Path), logger.LogError)
		}
	}

	// Report on updates
	if len(updates) > 0 {
		logger.Logger(fmt.Sprintf("Updated %d recipes:", len(updates)), logger.LogSuccess)
		for _, recipe := range updates {
			name, _ := recipe.Name()
			logger.Logger(fmt.Sprintf("  - %s to version %s", name, recipe.UpdatedVersion()), logger.LogSuccess)
		}
	}

	return nil
}
