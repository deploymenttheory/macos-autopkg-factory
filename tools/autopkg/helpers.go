package autopkg

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// Helper functions

// downloadFile downloads a file from the given URL to the specified path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// fileExists checks if a file exists
func fileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// readJSONFile reads a JSON file into a map
func readJSONFile(filepath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// writeJSONFile writes a map to a JSON file
func writeJSONFile(filepath string, data map[string]interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, jsonData, 0644)
}

// updateJSONFile updates a specific key in a JSON file
func updateJSONFile(filepath string, key string, value interface{}) error {
	// Read the current JSON
	data, err := readJSONFile(filepath)
	if err != nil {
		// If the file doesn't exist or can't be parsed, create a new map
		if os.IsNotExist(err) || err.Error() == "unexpected end of JSON input" {
			data = make(map[string]interface{})
		} else {
			return err
		}
	}

	// Update the key
	data[key] = value

	// Write the updated JSON back to the file
	return writeJSONFile(filepath, data)
}
