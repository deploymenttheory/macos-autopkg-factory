// recipe_parser.go
package autopkg

import (
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

// NewParserFromInput creates a parser based on provided input type.
func NewParserFromInput(input string) *RecipeParser {
	if input == "" {
		return &RecipeParser{source: &EnvironmentRecipeSource{EnvVarName: "RUN_RECIPE"}}
	}

	if _, err := os.Stat(input); err == nil && strings.HasSuffix(strings.ToLower(input), ".txt") {
		return &RecipeParser{source: &FileRecipeSource{FilePath: input}}
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
