package autopkg

import (
	"fmt"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// AutoPkgOrchestrator provides a fluent interface for building and executing AutoPkg pipelines
// It uses the existing functions from the autopkg package to execute the actual operations
type AutoPkgOrchestrator struct {
	steps   []PipelineExecutionStep
	options *PipelineOptions
}

// NewAutoPkgOrchestrator creates a new orchestrator with default options
func NewAutoPkgOrchestrator() *AutoPkgOrchestrator {
	return &AutoPkgOrchestrator{
		steps: []PipelineExecutionStep{},
		options: &PipelineOptions{
			MaxConcurrent:    4,
			Timeout:          60 * time.Minute,
			StopOnFirstError: false,
		},
	}
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

// WithStopOnFirstError configures the pipeline to stop on the first error
func (o *AutoPkgOrchestrator) WithStopOnFirstError(stop bool) *AutoPkgOrchestrator {
	o.options.StopOnFirstError = stop
	return o
}

// WithReportFile sets the path for the pipeline report file
func (o *AutoPkgOrchestrator) WithReportFile(reportFile string) *AutoPkgOrchestrator {
	o.options.ReportFile = reportFile
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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

	o.steps = append(o.steps, PipelineExecutionStep{
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
	o.steps = append(o.steps, PipelineExecutionStep{
		Type:            "repo-list",
		Recipes:         []string{},
		Options:         o.options.PrefsPath,
		ContinueOnError: continueOnError,
		Name:            "Repository List",
		Description:     "List all installed repositories",
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

	o.steps = append(o.steps, PipelineExecutionStep{
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
	o.steps = append(o.steps, PipelineExecutionStep{
		Type:            "repo-update",
		Recipes:         repos,
		Options:         o.options.PrefsPath,
		ContinueOnError: continueOnError,
		Name:            "Update Repositories",
		Description:     fmt.Sprintf("Update %d repositories", len(repos)),
	})
	return o
}

// AddFilterStep adds a recipe filtering step
// Uses FilterRecipes under the hood
func (o *AutoPkgOrchestrator) AddFilterStep(criteria *RecipeFilterCriteria, continueOnError bool) *AutoPkgOrchestrator {
	if criteria == nil {
		criteria = &RecipeFilterCriteria{
			IncludeOverrides: true,
		}
	}

	o.steps = append(o.steps, PipelineExecutionStep{
		Type:            "filter",
		Recipes:         []string{},
		Options:         criteria,
		ContinueOnError: continueOnError,
		Name:            "Recipe Filter",
		Description:     "Filter recipes based on criteria",
	})
	return o
}

// AddConditionalStep adds a step that only runs if a condition is met
func (o *AutoPkgOrchestrator) AddConditionalStep(step PipelineExecutionStep, condition func() bool) *AutoPkgOrchestrator {
	step.Condition = condition
	o.steps = append(o.steps, step)
	return o
}

// AddCustomStep adds a custom step with full control over parameters
func (o *AutoPkgOrchestrator) AddCustomStep(stepType, name, description string, recipes []string, options interface{}, continueOnError bool) *AutoPkgOrchestrator {
	o.steps = append(o.steps, PipelineExecutionStep{
		Type:            stepType,
		Name:            name,
		Description:     description,
		Recipes:         recipes,
		Options:         options,
		ContinueOnError: continueOnError,
	})
	return o
}

// Build returns the configured pipeline steps and options
func (o *AutoPkgOrchestrator) Build() ([]PipelineExecutionStep, *PipelineOptions) {
	return o.steps, o.options
}

// Validate checks if the pipeline configuration is valid
func (o *AutoPkgOrchestrator) Validate() error {
	if len(o.steps) == 0 {
		return fmt.Errorf("pipeline must contain at least one step")
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

// Execute builds, validates and executes the pipeline
// This calls the existing AutopkgPipeline function with the configured steps
func (o *AutoPkgOrchestrator) Execute() (*PipelineResult, error) {
	logger.Logger("🔍 Validating pipeline configuration", logger.LogInfo)
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("pipeline validation failed: %w", err)
	}

	logger.Logger("🚀 Executing pipeline with AutopkgPipeline", logger.LogInfo)
	steps, options := o.Build()
	return AutopkgPipeline(steps, options)
}

// ExecuteWithContext allows for pre/post execution hooks
func (o *AutoPkgOrchestrator) ExecuteWithContext(preExecHook func(), postExecHook func(*PipelineResult, error)) (*PipelineResult, error) {
	// Run pre-execution hook if provided
	if preExecHook != nil {
		preExecHook()
	}

	// Execute the pipeline
	result, err := o.Execute()

	// Run post-execution hook if provided
	if postExecHook != nil {
		postExecHook(result, err)
	}

	return result, err
}
