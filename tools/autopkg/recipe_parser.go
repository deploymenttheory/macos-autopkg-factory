// recipe_parser.go
package autopkg

import (
	"encoding/json"
	"os"
	"strings"
)

// RecipeSource defines an interface for different recipe sources
type RecipeSource interface {
	GetRecipes() ([]string, error)
}

// EnvironmentRecipeSource implementation
type EnvironmentRecipeSource struct {
	EnvVarName string
}

func (s *EnvironmentRecipeSource) GetRecipes() ([]string, error) {
	envRecipes := os.Getenv(s.EnvVarName)
	if envRecipes == "" {
		return nil, nil
	}
	recipes := strings.Split(envRecipes, ",")
	return normalizeRecipeNames(recipes), nil
}

// FileRecipeSource implementation (expects a .txt file)
type FileRecipeSource struct {
	FilePath string
}

func (s *FileRecipeSource) GetRecipes() ([]string, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var recipes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			recipes = append(recipes, line)
		}
	}
	return normalizeRecipeNames(recipes), nil
}

// CommaDelimitedRecipeSource implementation
type CommaDelimitedRecipeSource struct {
	Input string
}

func (s *CommaDelimitedRecipeSource) GetRecipes() ([]string, error) {
	rawRecipes := strings.Split(s.Input, ",")
	return normalizeRecipeNames(rawRecipes), nil
}

// DirectRecipeSource implementation
type DirectRecipeSource struct {
	Recipes []string
}

func (s *DirectRecipeSource) GetRecipes() ([]string, error) {
	return normalizeRecipeNames(s.Recipes), nil
}

// RecipeParser handles input normalization for recipe execution.
type RecipeParser struct {
	source RecipeSource
}

// Add a new JSONFileRecipeSource implementation
type JSONFileRecipeSource struct {
	FilePath string
}

func (s *JSONFileRecipeSource) GetRecipes() ([]string, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, err
	}

	var recipes []string
	if err := json.Unmarshal(data, &recipes); err != nil {
		return nil, err
	}

	return normalizeRecipeNames(recipes), nil
}

// ParseRecipeInput parses recipe input types and prepares them for autopkg
// supported scenarios are single recipe as an env var or from a cli flag
// a comma delimited list of recipes, or recipes stored in a list.
func ParseRecipeInput(input string) *RecipeParser {
	if input == "" {
		return &RecipeParser{source: &EnvironmentRecipeSource{EnvVarName: "RUN_RECIPE"}}
	}

	if _, err := os.Stat(input); err == nil {
		// File exists, check extension
		inputLower := strings.ToLower(input)
		if strings.HasSuffix(inputLower, ".txt") {
			return &RecipeParser{source: &FileRecipeSource{FilePath: input}}
		} else if strings.HasSuffix(inputLower, ".json") {
			return &RecipeParser{source: &JSONFileRecipeSource{FilePath: input}}
		}
	}

	if strings.Contains(input, ",") {
		return &RecipeParser{source: &CommaDelimitedRecipeSource{Input: input}}
	}

	return &RecipeParser{source: &DirectRecipeSource{Recipes: []string{input}}}
}

// Parse returns a normalized slice of recipe names.
func (rp *RecipeParser) Parse() ([]string, error) {
	return rp.source.GetRecipes()
}

// normalizeRecipeNames normalizes recipe names by trimming whitespace and appending .recipe if missing
func normalizeRecipeNames(recipes []string) []string {
	normalized := make([]string, 0, len(recipes))
	for _, recipe := range recipes {
		recipe = strings.TrimSpace(recipe)
		if !strings.HasSuffix(recipe, ".recipe") {
			recipe += ".recipe"
		}
		normalized = append(normalized, recipe)
	}
	return normalized
}
