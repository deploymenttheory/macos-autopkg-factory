// orchestrator.go contains workflow functions for composing autopkg with steps taken from run.go
package autopkg

import (
	"fmt"
	"os"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// WorkflowStep defines a step in the autopkg workflow
type WorkflowStep struct {
	Type            string      // Type of step: "verify", "update-trust", "run", "cleanup", etc.
	Recipes         []string    // Recipes to process in this step
	Options         interface{} // Step-specific options
	ContinueOnError bool        // Whether to continue if this step fails
	Name            string      // Optional name for the step
	Description     string      // Optional description
	Condition       func() bool // Optional condition to determine if this step should run
}

// WorkflowOptions contains options for the entire workflow
type WorkflowOptions struct {
	PrefsPath          string
	MaxConcurrent      int
	Timeout            time.Duration
	StopOnFirstError   bool
	ReportFile         string
	NotifyOnCompletion bool
	NotifyOnError      bool
	WebhookURL         string
}

// WorkflowResult contains the results of the workflow execution
type WorkflowResult struct {
	Success          bool
	FailedSteps      []string
	CompletedSteps   []string
	SkippedSteps     []string
	ProcessedRecipes map[string]bool
	Errors           map[string]error
	StartTime        time.Time
	EndTime          time.Time
	ElapsedTime      time.Duration
}

// AutoPkgOrchestrator provides a fluent interface for building and executing AutoPkg workflows
type AutoPkgOrchestrator struct {
	steps   []WorkflowStep
	options *WorkflowOptions
}

// NewAutoPkgOrchestrator creates a new orchestrator with default options
func NewAutoPkgOrchestrator() *AutoPkgOrchestrator {
	return &AutoPkgOrchestrator{
		steps: []WorkflowStep{},
		options: &WorkflowOptions{
			MaxConcurrent:    4,
			Timeout:          60 * time.Minute,
			StopOnFirstError: false,
		},
	}
}

// AddRootCheckStep adds a step to verify the script is not running as root
func (o *AutoPkgOrchestrator) AddRootCheckStep(continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "root-check",
		Recipes:         []string{},
		Options:         nil,
		ContinueOnError: continueOnError,
		Name:            "Root User Check",
		Description:     "Verify script is not running as root user",
	})
	return o
}

// AddInstallAutoPkgStep adds a step to ensure AutoPkg is installed and up to date.
// If AutoPkg is already installed, it verifies the existing version.
// If 'ForceUpdate' is enabled in the InstallConfig, it will perform a forced update;
// otherwise, it will skip installation.
func (o *AutoPkgOrchestrator) AddInstallAutoPkgStep(installConfig *InstallConfig, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "install-autopkg",
		Recipes:         []string{},
		Options:         installConfig,
		ContinueOnError: continueOnError,
		Name:            "Ensure AutoPkg Installed",
		Description:     "Check for AutoPkg installation, verify version, and update if required",
	})
	return o
}

// AddGitCheckStep adds a step to check/install Git
func (o *AutoPkgOrchestrator) AddGitCheckStep(continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "check-git",
		Recipes:         []string{},
		Options:         nil,
		ContinueOnError: continueOnError,
		Name:            "Git Check",
		Description:     "Check and install Git if needed",
	})
	return o
}

// WithPrefsPath sets the preferences path for all operations
func (o *AutoPkgOrchestrator) WithPrefsPath(prefsPath string) *AutoPkgOrchestrator {
	o.options.PrefsPath = prefsPath
	return o
}

// WithConcurrency sets the maximum concurrent recipes for parallel operations
func (o *AutoPkgOrchestrator) WithConcurrency(max int) *AutoPkgOrchestrator {
	o.options.MaxConcurrent = max
	return o
}

// WithTimeout sets the timeout for parallel operations
func (o *AutoPkgOrchestrator) WithTimeout(timeout time.Duration) *AutoPkgOrchestrator {
	o.options.Timeout = timeout
	return o
}

// WithStopOnFirstError configures the workflow to stop on the first error
func (o *AutoPkgOrchestrator) WithStopOnFirstError(stop bool) *AutoPkgOrchestrator {
	o.options.StopOnFirstError = stop
	return o
}

// WithReportFile sets the path for the workflow report file
func (o *AutoPkgOrchestrator) WithReportFile(reportFile string) *AutoPkgOrchestrator {
	o.options.ReportFile = reportFile
	return o
}

// AddRepoAddStep adds a step to add repositories
// Uses AddRepo under the hood
func (o *AutoPkgOrchestrator) AddRepoAddStep(repoURLs []string, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "repo-add",
		Recipes:         repoURLs,
		Options:         o.options.PrefsPath,
		ContinueOnError: continueOnError,
		Name:            "Add Repositories",
		Description:     fmt.Sprintf("Add %d repositories", len(repoURLs)),
	})
	return o
}

// AddRecipeRepoAnalysisStep adds a step to analyze recipe dependencies and add required repositories
// This step analyzes the specified recipes, determines which repositories are needed,
// and optionally adds them to AutoPkg automatically.
func (o *AutoPkgOrchestrator) AddRecipeRepoAnalysisStep(
	recipes []string,
	options *RecipeRepoAnalysisOptions,
	addRepos bool,
	continueOnError bool) *AutoPkgOrchestrator {

	if options == nil {
		options = &RecipeRepoAnalysisOptions{
			RecipeIdentifiers: recipes,
			IncludeParents:    true,
			MaxDepth:          5,
			VerifyRepoExists:  true,
			PrefsPath:         o.options.PrefsPath,
			IncludeBase:       true,
		}
	} else {
		// Ensure the recipe identifiers are set
		options.RecipeIdentifiers = recipes
		// Use the orchestrator prefs path if not specified
		if options.PrefsPath == "" {
			options.PrefsPath = o.options.PrefsPath
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:    "repo-analysis",
		Recipes: recipes,
		Options: &struct {
			AnalysisOptions *RecipeRepoAnalysisOptions
			AddRepos        bool
		}{
			AnalysisOptions: options,
			AddRepos:        addRepos,
		},
		ContinueOnError: continueOnError,
		Name:            "Recipe Repository Analysis",
		Description:     fmt.Sprintf("Analyze %d recipes for required repositories", len(recipes)),
	})

	return o
}

// WithWebhookNotifications enables webhook notifications
func (o *AutoPkgOrchestrator) WithWebhookNotifications(url string, notifyOnError, notifyOnCompletion bool) *AutoPkgOrchestrator {
	o.options.WebhookURL = url
	o.options.NotifyOnError = notifyOnError
	o.options.NotifyOnCompletion = notifyOnCompletion
	return o
}

// AddVerifyStep adds a trust verification step
// Uses VerifyTrustInfoForRecipes under the hood
func (o *AutoPkgOrchestrator) AddVerifyStep(recipes []string, options *VerifyTrustInfoOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &VerifyTrustInfoOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "verify",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Trust Verification",
		Description:     fmt.Sprintf("Verify trust info for %d recipes", len(recipes)),
	})
	return o
}

// AddUpdateTrustStep adds a trust update step
// Uses UpdateTrustInfoForRecipes under the hood
func (o *AutoPkgOrchestrator) AddUpdateTrustStep(recipes []string, options *UpdateTrustInfoOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &UpdateTrustInfoOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "update-trust",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Trust Update",
		Description:     fmt.Sprintf("Update trust info for %d recipes", len(recipes)),
	})
	return o
}

// AddRunStep adds a recipe run step
// Uses RunRecipes under the hood
func (o *AutoPkgOrchestrator) AddRunStep(recipes []string, options *RunOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &RunOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "run",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe Run",
		Description:     fmt.Sprintf("Run %d recipes", len(recipes)),
	})
	return o
}

// AddParallelRunStep adds a parallel recipe run step
// Uses ParallelRunRecipes under the hood
func (o *AutoPkgOrchestrator) AddParallelRunStep(recipes []string, options *ParallelRunOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &ParallelRunOptions{
			PrefsPath:     o.options.PrefsPath,
			MaxConcurrent: o.options.MaxConcurrent,
			Timeout:       o.options.Timeout,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "parallel-run",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Parallel Recipe Run",
		Description:     fmt.Sprintf("Run %d recipes in parallel", len(recipes)),
	})
	return o
}

// AddBatchProcessingStep adds a batch processing step
// Uses RecipeBatchProcessing under the hood
func (o *AutoPkgOrchestrator) AddBatchProcessingStep(recipes []string, options *RecipeBatchOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &RecipeBatchOptions{
			PrefsPath:            o.options.PrefsPath,
			MaxConcurrentRecipes: o.options.MaxConcurrent,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "batch",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Batch Processing",
		Description:     fmt.Sprintf("Process %d recipes in batch", len(recipes)),
	})
	return o
}

// AddCleanupStep adds a cache cleanup step
// Uses CleanupCache under the hood
func (o *AutoPkgOrchestrator) AddCleanupStep(options *CleanupOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &CleanupOptions{
			PrefsPath:         o.options.PrefsPath,
			RemoveDownloads:   true,
			RemoveRecipeCache: true,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "cleanup",
		Recipes:         []string{},
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Cache Cleanup",
		Description:     "Clean up AutoPkg cache",
	})
	return o
}

// AddValidateStep adds a recipe validation step
// Uses ValidateRecipeList under the hood
func (o *AutoPkgOrchestrator) AddValidateStep(recipes []string, options *ValidateRecipeListOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &ValidateRecipeListOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "validate",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe Validation",
		Description:     fmt.Sprintf("Validate %d recipes", len(recipes)),
	})
	return o
}

// AddImportStep adds a repo import step
// Uses ImportRecipesFromRepo under the hood
func (o *AutoPkgOrchestrator) AddImportStep(repoURL string, options *ImportRecipesFromRepoOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &ImportRecipesFromRepoOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "import",
		Recipes:         []string{repoURL},
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Repo Import",
		Description:     fmt.Sprintf("Import recipes from %s", repoURL),
	})
	return o
}

// AddAuditStep adds a recipe audit step
// Uses AuditRecipe under the hood
func (o *AutoPkgOrchestrator) AddAuditStep(recipes []string, options *AuditOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &AuditOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "audit",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe Audit",
		Description:     fmt.Sprintf("Audit %d recipes", len(recipes)),
	})
	return o
}

// AddInstallStep adds a recipe installation step
// Uses InstallRecipe under the hood
func (o *AutoPkgOrchestrator) AddInstallStep(recipes []string, options *InstallOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &InstallOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "install",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe Installation",
		Description:     fmt.Sprintf("Install %d recipes", len(recipes)),
	})
	return o
}

// AddSearchStep adds a recipe search step
// Uses SearchRecipes under the hood
func (o *AutoPkgOrchestrator) AddSearchStep(searchTerm string, options *SearchOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &SearchOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "search",
		Recipes:         []string{searchTerm},
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe Search",
		Description:     fmt.Sprintf("Search for recipes matching '%s'", searchTerm),
	})
	return o
}

// AddListStep adds a recipe listing step
// Uses ListRecipes under the hood
func (o *AutoPkgOrchestrator) AddListStep(options *ListRecipeOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &ListRecipeOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "list",
		Recipes:         []string{},
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Recipe List",
		Description:     "List all available recipes",
	})
	return o
}

// AddRepoListStep adds a repository listing step
// Uses ListRepos under the hood
func (o *AutoPkgOrchestrator) AddRepoListStep(continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "repo-list",
		Recipes:         []string{},
		Options:         o.options.PrefsPath,
		ContinueOnError: continueOnError,
		Name:            "Repository List",
		Description:     "List all installed repositories",
	})
	return o
}

// AddSetPreferencesStep adds a step to configure AutoPkg preferences
func (o *AutoPkgOrchestrator) AddSetPreferencesStep(prefs *PreferencesData, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "set-preferences",
		Recipes:         []string{},
		Options:         prefs,
		ContinueOnError: continueOnError,
		Name:            "Configure Preferences",
		Description:     "Set AutoPkg preferences",
	})
	return o
}

// AddMakeOverrideStep adds a step to create recipe overrides
// Uses MakeOverride under the hood
func (o *AutoPkgOrchestrator) AddMakeOverrideStep(recipes []string, options *MakeOverrideOptions, continueOnError bool) *AutoPkgOrchestrator {
	if options == nil {
		options = &MakeOverrideOptions{
			PrefsPath: o.options.PrefsPath,
		}
	}

	o.steps = append(o.steps, WorkflowStep{
		Type:            "make-override",
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
		Name:            "Create Overrides",
		Description:     fmt.Sprintf("Create overrides for %d recipes", len(recipes)),
	})
	return o
}

// AddRepoUpdateStep adds a repository update step
// Uses UpdateRepo under the hood
func (o *AutoPkgOrchestrator) AddRepoUpdateStep(repos []string, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            "repo-update",
		Recipes:         repos,
		Options:         o.options.PrefsPath,
		ContinueOnError: continueOnError,
		Name:            "Update Repositories",
		Description:     fmt.Sprintf("Update %d repositories", len(repos)),
	})
	return o
}

// AddConditionalStep adds a step that only runs if a condition is met
func (o *AutoPkgOrchestrator) AddConditionalStep(step WorkflowStep, condition func() bool) *AutoPkgOrchestrator {
	step.Condition = condition
	o.steps = append(o.steps, step)
	return o
}

// AddCustomStep adds a custom step with full control over parameters
func (o *AutoPkgOrchestrator) AddCustomStep(stepType, name, description string, recipes []string, options interface{}, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, WorkflowStep{
		Type:            stepType,
		Name:            name,
		Description:     description,
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
	})
	return o
}

// Validate checks if the workflow configuration is valid
func (o *AutoPkgOrchestrator) Validate() error {
	if len(o.steps) == 0 {
		return fmt.Errorf("workflow must contain at least one step")
	}

	for i, step := range o.steps {
		switch step.Type {
		case "import", "search":
			if len(step.Recipes) != 1 {
				return fmt.Errorf("step %d (%s): %s step requires exactly one value", i, step.Name, step.Type)
			}
		case "list", "repo-list", "cleanup", "filter":
			// These don't require recipes, so no validation needed
		case "run", "parallel-run", "batch", "verify", "update-trust", "validate", "make-override", "install", "audit", "repo-update":
			if len(step.Recipes) == 0 {
				return fmt.Errorf("step %d (%s): %s step requires at least one recipe", i, step.Name, step.Type)
			}
		}
	}

	return nil
}

// Execute builds, validates and executes the workflow
// This directly executes each step without calling a separate pipeline function
func (o *AutoPkgOrchestrator) Execute() (*WorkflowResult, error) {
	logger.Logger("🔍 Validating workflow configuration", logger.LogInfo)
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	logger.Logger("🚀 Starting AutoPkg workflow execution", logger.LogInfo)

	result := &WorkflowResult{
		Success:          true,
		FailedSteps:      []string{},
		CompletedSteps:   []string{},
		SkippedSteps:     []string{},
		ProcessedRecipes: make(map[string]bool),
		Errors:           make(map[string]error),
		StartTime:        time.Now(),
	}

	// Execute each step in sequence
	for i, step := range o.steps {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d (%s)", i+1, step.Type)
		}

		// Check if this step should be run
		if step.Condition != nil && !step.Condition() {
			logger.Logger(fmt.Sprintf("⏭️ Skipping step %s: condition not met", stepName), logger.LogInfo)
			result.SkippedSteps = append(result.SkippedSteps, stepName)
			continue
		}

		logger.Logger(fmt.Sprintf("▶️ Executing step %s: %s", stepName, step.Description), logger.LogInfo)

		var stepErr error
		switch step.Type {
		case "root-check":
			err := RootCheck()
			if err != nil {
				stepErr = fmt.Errorf("root check failed: %w", err)
			} else {
				logger.Logger("✅ Root check passed - not running as root", logger.LogSuccess)
			}
			// In the Execute method, add these cases to the switch statement:
		case "install-autopkg":
			config, ok := step.Options.(*InstallConfig)
			if !ok {
				stepErr = fmt.Errorf("invalid options for install-autopkg step")
				break
			}

			version, err := InstallAutoPkg(config)
			if err != nil {
				stepErr = fmt.Errorf("AutoPkg installation failed: %w", err)
			} else {
				logger.Logger(fmt.Sprintf("✅ AutoPkg %s installed successfully", version), logger.LogSuccess)
			}

		case "check-git":
			err := CheckGit()
			if err != nil {
				stepErr = fmt.Errorf("Git check/installation failed: %w", err)
			}

		case "set-preferences":
			prefs, ok := step.Options.(*PreferencesData)
			if !ok {
				stepErr = fmt.Errorf("invalid options for set-preferences step")
				break
			}

			err := SetAutoPkgPreferences(o.options.PrefsPath, prefs)
			if err != nil {
				stepErr = fmt.Errorf("failed to set preferences: %w", err)
			}

		case "verify":
			verifyOptions, ok := step.Options.(*VerifyTrustInfoOptions)
			if !ok {
				verifyOptions = &VerifyTrustInfoOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			success, failedRecipes, err := VerifyTrustInfoForRecipes(step.Recipes, verifyOptions)
			if err != nil || !success {
				stepErr = fmt.Errorf("trust verification failed for %d recipes", len(failedRecipes))
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "update-trust":
			updateOptions, ok := step.Options.(*UpdateTrustInfoOptions)
			if !ok {
				updateOptions = &UpdateTrustInfoOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := UpdateTrustInfoForRecipes(step.Recipes, updateOptions)
			if err != nil {
				stepErr = fmt.Errorf("trust update failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "repo-analysis":
			options, ok := step.Options.(*struct {
				AnalysisOptions *RecipeRepoAnalysisOptions
				AddRepos        bool
			})
			if !ok {
				stepErr = fmt.Errorf("invalid options for repo-analysis step")
				break
			}

			// Analyze the recipes for repository dependencies
			dependencies, err := AnalyzeRecipeRepoDependencies(options.AnalysisOptions)
			if err != nil {
				stepErr = fmt.Errorf("recipe repository analysis failed: %w", err)
				break
			}

			// Log the dependencies
			logger.Logger(fmt.Sprintf("🔍 Found %d repository dependencies for %d recipes",
				len(dependencies), len(step.Recipes)), logger.LogInfo)

			for _, dep := range dependencies {
				logger.Logger(fmt.Sprintf("  - %s: %s", dep.RecipeIdentifier, dep.RepoURL), logger.LogInfo)
			}

			// If requested, add the repositories
			if options.AddRepos {
				var repoURLs []string
				for _, dep := range dependencies {
					repoURLs = append(repoURLs, dep.RepoURL)
				}

				if len(repoURLs) > 0 {
					// Remove duplicates
					uniqueURLs := make(map[string]bool)
					var filteredURLs []string
					for _, url := range repoURLs {
						if !uniqueURLs[url] {
							uniqueURLs[url] = true
							filteredURLs = append(filteredURLs, url)
						}
					}

					logger.Logger(fmt.Sprintf("⬇️ Adding %d unique repositories", len(filteredURLs)), logger.LogInfo)
					if err := AddRepo(filteredURLs, options.AnalysisOptions.PrefsPath); err != nil {
						stepErr = fmt.Errorf("failed to add repositories: %w", err)
						break
					}
				}
			}

		case "repo-add":
			prefsPath, ok := step.Options.(string)
			if !ok {
				prefsPath = o.options.PrefsPath
			}

			err := AddRepo(step.Recipes, prefsPath)
			if err != nil {
				stepErr = fmt.Errorf("add repos failed: %w", err)
			}

		case "run":
			runOptions, ok := step.Options.(*RunOptions)
			if !ok {
				runOptions = &RunOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := RunRecipes(step.Recipes, runOptions)
			if err != nil {
				stepErr = fmt.Errorf("run recipes failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "parallel-run":
			parallelOptions, ok := step.Options.(*ParallelRunOptions)
			if !ok {
				parallelOptions = &ParallelRunOptions{
					PrefsPath:     o.options.PrefsPath,
					MaxConcurrent: o.options.MaxConcurrent,
					Timeout:       o.options.Timeout,
				}
			}

			_, err := ParallelRunRecipes(step.Recipes, parallelOptions)
			if err != nil {
				stepErr = fmt.Errorf("parallel run failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "batch":
			batchOptions, ok := step.Options.(*RecipeBatchOptions)
			if !ok {
				batchOptions = &RecipeBatchOptions{
					PrefsPath:            o.options.PrefsPath,
					MaxConcurrentRecipes: o.options.MaxConcurrent,
				}
			}

			_, err := RecipeBatchProcessing(step.Recipes, batchOptions)
			if err != nil {
				stepErr = fmt.Errorf("batch processing failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "cleanup":
			cleanupOptions, ok := step.Options.(*CleanupOptions)
			if !ok {
				cleanupOptions = &CleanupOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := CleanupCache(cleanupOptions)
			if err != nil {
				stepErr = fmt.Errorf("cleanup failed: %w", err)
			}

		case "validate":
			validateOptions, ok := step.Options.(*ValidateRecipeListOptions)
			if !ok {
				validateOptions = &ValidateRecipeListOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			_, err := ValidateRecipeList(step.Recipes, validateOptions)
			if err != nil {
				stepErr = fmt.Errorf("validation failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "import":
			if len(step.Recipes) != 1 {
				stepErr = fmt.Errorf("import step requires exactly one repo URL")
				break
			}

			importOptions, ok := step.Options.(*ImportRecipesFromRepoOptions)
			if !ok {
				importOptions = &ImportRecipesFromRepoOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			importedRecipes, err := ImportRecipesFromRepo(step.Recipes[0], importOptions)
			if err != nil {
				stepErr = fmt.Errorf("import failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range importedRecipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "audit":
			auditOptions, ok := step.Options.(*AuditOptions)
			if !ok {
				auditOptions = &AuditOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := AuditRecipe(step.Recipes, auditOptions)
			if err != nil {
				stepErr = fmt.Errorf("audit failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "install":
			installOptions, ok := step.Options.(*InstallOptions)
			if !ok {
				installOptions = &InstallOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := InstallRecipe(step.Recipes, installOptions)
			if err != nil {
				stepErr = fmt.Errorf("install failed: %w", err)
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "search":
			if len(step.Recipes) != 1 {
				stepErr = fmt.Errorf("search step requires exactly one search term")
				break
			}

			searchOptions, ok := step.Options.(*SearchOptions)
			if !ok {
				searchOptions = &SearchOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := SearchRecipes(step.Recipes[0], searchOptions)
			if err != nil {
				stepErr = fmt.Errorf("search failed: %w", err)
			}

		case "list":
			listOptions, ok := step.Options.(*ListRecipeOptions)
			if !ok {
				listOptions = &ListRecipeOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			err := ListRecipes(listOptions)
			if err != nil {
				stepErr = fmt.Errorf("list recipes failed: %w", err)
			}

		case "repo-list":
			prefsPath, ok := step.Options.(string)
			if !ok {
				prefsPath = o.options.PrefsPath
			}

			err := ListRepos(prefsPath)
			if err != nil {
				stepErr = fmt.Errorf("list repos failed: %w", err)
			}

		case "make-override":
			if len(step.Recipes) == 0 {
				stepErr = fmt.Errorf("make-override step requires at least one recipe")
				break
			}

			overrideOptions, ok := step.Options.(*MakeOverrideOptions)
			if !ok {
				overrideOptions = &MakeOverrideOptions{
					PrefsPath: o.options.PrefsPath,
				}
			}

			for _, recipe := range step.Recipes {
				err := MakeOverride(recipe, overrideOptions)
				if err != nil {
					stepErr = fmt.Errorf("make override for %s failed: %w", recipe, err)
					break
				}
			}

			// Mark processed recipes
			for _, recipe := range step.Recipes {
				result.ProcessedRecipes[recipe] = true
			}

		case "repo-update":
			prefsPath, ok := step.Options.(string)
			if !ok {
				prefsPath = o.options.PrefsPath
			}

			err := UpdateRepo(step.Recipes, prefsPath)
			if err != nil {
				stepErr = fmt.Errorf("repo update failed: %w", err)
			}

		default:
			stepErr = fmt.Errorf("unknown step type: %s", step.Type)
		}

		if stepErr != nil {
			logger.Logger(fmt.Sprintf("❌ Step %s failed: %v", stepName, stepErr), logger.LogError)
			result.Errors[stepName] = stepErr
			result.FailedSteps = append(result.FailedSteps, stepName)
			result.Success = false

			if o.options.NotifyOnError {
				// Send error notification if configured
				if o.options.WebhookURL != "" {
					notifyErr := sendWebhookNotification(o.options.WebhookURL, fmt.Sprintf("Workflow step '%s' failed: %v", stepName, stepErr))
					if notifyErr != nil {
						logger.Logger(fmt.Sprintf("⚠️ Failed to send error notification: %v", notifyErr), logger.LogWarning)
					}
				}
			}

			if o.options.StopOnFirstError {
				break
			}
		} else {
			logger.Logger(fmt.Sprintf("✅ Step %s completed successfully", stepName), logger.LogSuccess)
			result.CompletedSteps = append(result.CompletedSteps, stepName)
		}
	}

	// Calculate elapsed time
	result.EndTime = time.Now()
	result.ElapsedTime = result.EndTime.Sub(result.StartTime)

	// Generate report if requested
	if o.options.ReportFile != "" {
		reportData := fmt.Sprintf("AutoPkg Workflow Report\n"+
			"=====================\n\n"+
			"Start time: %s\n"+
			"End time: %s\n"+
			"Elapsed time: %s\n\n"+
			"Success: %t\n"+
			"Completed steps: %d\n"+
			"Failed steps: %d\n"+
			"Skipped steps: %d\n"+
			"Processed recipes: %d\n\n",
			result.StartTime.Format(time.RFC3339),
			result.EndTime.Format(time.RFC3339),
			result.ElapsedTime,
			result.Success,
			len(result.CompletedSteps),
			len(result.FailedSteps),
			len(result.SkippedSteps),
			len(result.ProcessedRecipes))

		// Add details about failed steps
		if len(result.FailedSteps) > 0 {
			reportData += "Failed Steps:\n"
			for _, stepName := range result.FailedSteps {
				reportData += fmt.Sprintf("- %s: %v\n", stepName, result.Errors[stepName])
			}
			reportData += "\n"
		}

		if err := os.WriteFile(o.options.ReportFile, []byte(reportData), 0644); err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to write report file: %v", err), logger.LogWarning)
		} else {
			logger.Logger(fmt.Sprintf("📊 Workflow report written to %s", o.options.ReportFile), logger.LogInfo)
		}
	}

	// Send completion notification if configured
	if o.options.NotifyOnCompletion && o.options.WebhookURL != "" {
		status := "succeeded"
		if !result.Success {
			status = "failed"
		}

		notification := fmt.Sprintf("AutoPkg Workflow %s after %s. Processed %d recipes, %d steps completed, %d steps failed.",
			status, result.ElapsedTime, len(result.ProcessedRecipes), len(result.CompletedSteps), len(result.FailedSteps))

		if err := sendWebhookNotification(o.options.WebhookURL, notification); err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to send completion notification: %v", err), logger.LogWarning)
		}
	}

	logger.Logger(fmt.Sprintf("🏁 Workflow execution completed in %s: %d steps completed, %d steps failed, %d steps skipped",
		result.ElapsedTime, len(result.CompletedSteps), len(result.FailedSteps), len(result.SkippedSteps)), logger.LogInfo)

	if !result.Success {
		return result, fmt.Errorf("workflow execution failed: %d steps failed", len(result.FailedSteps))
	}

	return result, nil
}
