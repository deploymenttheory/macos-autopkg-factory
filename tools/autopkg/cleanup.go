package autopkg

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// CleanupOptions contains options for cleaning up the AutoPkg cache
type CleanupOptions struct {
	PrefsPath         string
	RemoveDownloads   bool
	RemoveRecipeCache bool
	KeepDays          int
}

// CleanupCache cleans up AutoPkg's cache directories
func CleanupCache(options *CleanupOptions) error {
	if options == nil {
		options = &CleanupOptions{
			RemoveDownloads:   true,
			RemoveRecipeCache: true,
			KeepDays:          0, // 0 means all
		}
	}

	logger.Logger("🧹 Cleaning up AutoPkg cache", logger.LogInfo)

	// Determine cache directory
	cacheDir := ""
	if options.PrefsPath != "" {
		// Try to read from preferences for custom cache location
		prefs, err := GetAutoPkgPreferences(options.PrefsPath)
		if err == nil && prefs.AdditionalPreferences != nil {
			if cachePath, ok := prefs.AdditionalPreferences["CACHE_DIR"].(string); ok {
				cacheDir = cachePath
			}
		}
	}

	if cacheDir == "" {
		// Use default cache location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, "Library/AutoPkg/Cache")
	}

	// Ensure cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return fmt.Errorf("cache directory does not exist: %s", cacheDir)
	}

	// Get current time for age comparison
	now := time.Now()

	// Function to clean a directory based on age
	cleanDirectory := func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			entryPath := filepath.Join(dir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to get info for %s: %v", entryPath, err), logger.LogWarning)
				continue
			}

			// Check age if keepDays is specified
			if options.KeepDays > 0 {
				ageInDays := int(now.Sub(info.ModTime()).Hours() / 24)
				if ageInDays < options.KeepDays {
					// Skip files that are newer than the keepDays threshold
					continue
				}
			}

			if err := os.RemoveAll(entryPath); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to remove %s: %v", entryPath, err), logger.LogWarning)
			} else {
				logger.Logger(fmt.Sprintf("🗑️ Removed %s", entryPath), logger.LogInfo)
			}
		}
		return nil
	}

	// Clean downloads directory
	if options.RemoveDownloads {
		downloadsDir := filepath.Join(cacheDir, "downloads")
		if _, err := os.Stat(downloadsDir); err == nil {
			logger.Logger("🧹 Cleaning downloads cache", logger.LogInfo)
			if err := cleanDirectory(downloadsDir); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to clean downloads directory: %v", err), logger.LogWarning)
			}
		}
	}

	// Clean recipe cache directories
	if options.RemoveRecipeCache {
		logger.Logger("🧹 Cleaning recipe cache", logger.LogInfo)
		entries, err := os.ReadDir(cacheDir)
		if err != nil {
			return fmt.Errorf("failed to read cache directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "downloads" {
				recipeCacheDir := filepath.Join(cacheDir, entry.Name())
				if err := cleanDirectory(recipeCacheDir); err != nil {
					logger.Logger(fmt.Sprintf("⚠️ Failed to clean recipe cache %s: %v", entry.Name(), err), logger.LogWarning)
				}
			}
		}
	}

	logger.Logger("✅ AutoPkg cache cleanup completed", logger.LogSuccess)
	return nil
}
