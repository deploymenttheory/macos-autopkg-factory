// recipe_parser.go
package autopkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"howett.net/plist"
)

// RecipeSource defines an interface for different recipe sources
type RecipeSource interface {
	GetRecipes() ([]string, error)
}

// EnvironmentRecipeSource gets recipes from environment variables
type EnvironmentRecipeSource struct {
	EnvVarName string
}

// GetRecipes retrieves recipes from an environment variable
func (s *EnvironmentRecipeSource) GetRecipes() ([]string, error) {
	envRecipes := os.Getenv(s.EnvVarName)
	if envRecipes == "" {
		return nil, nil
	}

	logger.Logger(fmt.Sprintf("Using recipes from environment variable %s: %s", s.EnvVarName, envRecipes), logger.LogInfo)
	recipes := strings.Split(envRecipes, ", ")

	return normalizeRecipeNames(recipes), nil
}

// FileRecipeSource gets recipes from a file
type FileRecipeSource struct {
	FilePath string
}

// GetRecipes retrieves recipes from a file (JSON or plist)
func (s *FileRecipeSource) GetRecipes() ([]string, error) {
	if s.FilePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe list file: %w", err)
	}

	var recipes []string
	ext := strings.ToLower(filepath.Ext(s.FilePath))

	switch ext {
	case ".json":
		err = json.Unmarshal(data, &recipes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON recipe list: %w", err)
		}
	case ".plist":
		var recipeList struct {
			Recipes []string
		}
		_, err = plist.Unmarshal(data, &recipeList)
		if err != nil {
			return nil, fmt.Errorf("failed to parse plist recipe list: %w", err)
		}
		recipes = recipeList.Recipes
	default:
		return nil, fmt.Errorf("unsupported recipe list format: %s (expected .json or .plist)", ext)
	}

	logger.Logger(fmt.Sprintf("Parsed %d recipes from %s", len(recipes), s.FilePath), logger.LogInfo)
	return normalizeRecipeNames(recipes), nil
}

// DirectRecipeSource allows providing recipes directly
type DirectRecipeSource struct {
	Recipes []string
}

// GetRecipes returns the directly provided recipes
func (s *DirectRecipeSource) GetRecipes() ([]string, error) {
	return normalizeRecipeNames(s.Recipes), nil
}

// normalizeRecipeNames ensures all recipes have the .recipe extension
func normalizeRecipeNames(recipes []string) []string {
	result := make([]string, len(recipes))
	for i, recipe := range recipes {
		if !strings.HasSuffix(recipe, ".recipe") {
			result[i] = recipe + ".recipe"
		} else {
			result[i] = recipe
		}
	}
	return result
}

// RecipeParser manages the parsing process with multiple potential sources
type RecipeParser struct {
	sources []RecipeSource
}

// NewRecipeParser creates a new parser with the given sources
func NewRecipeParser(sources ...RecipeSource) *RecipeParser {
	return &RecipeParser{
		sources: sources,
	}
}

// ParseRecipes attempts to get recipes from all configured sources
func (p *RecipeParser) ParseRecipes() ([]string, error) {
	var allRecipes []string

	for _, source := range p.sources {
		recipes, err := source.GetRecipes()
		if err != nil {
			return nil, err
		}

		if len(recipes) > 0 {
			allRecipes = append(allRecipes, recipes...)
		}
	}

	if len(allRecipes) == 0 {
		return nil, fmt.Errorf("no recipes found from any configured source")
	}

	return allRecipes, nil
}

// RunRecipesFromList parses and runs recipes from list files and/or environment variables
func RunRecipesFromList(recipeListPath string, options *RecipeBatchRunOptions) (map[string]*RecipeBatchResult, error) {
	// Create a parser with file and environment sources
	parser := NewRecipeParser(
		&FileRecipeSource{FilePath: recipeListPath},
		&EnvironmentRecipeSource{EnvVarName: "RUN_RECIPE"},
	)

	// Parse recipes from all sources
	recipes, err := parser.ParseRecipes()
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipes: %w", err)
	}

	// Run the recipe batch processing
	logger.Logger(fmt.Sprintf("Processing %d recipes", len(recipes)), logger.LogInfo)

	return RunRecipeBatch(recipes, options)
}

// ParseRecipeList parses recipes from files and environment variables (for backward compatibility)
func ParseRecipeList(filePath string) ([]string, error) {
	parser := NewRecipeParser(
		&FileRecipeSource{FilePath: filePath},
		&EnvironmentRecipeSource{EnvVarName: "RUN_RECIPE"},
	)

	return parser.ParseRecipes()
}
