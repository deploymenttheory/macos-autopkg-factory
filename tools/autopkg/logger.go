package autopkg

import "fmt"

// Log levels
const (
	LogDebug   = 10
	LogInfo    = 20
	LogWarning = 30
	LogError   = 40
	LogSuccess = 50
)

// Global variables - can be set via environment variables
var (
	DEBUG         bool
	OVERRIDES_DIR string
	RECIPE_TO_RUN string
	TEAMS_WEBHOOK string
)

// Logger implements a simple logging system
func Logger(message string, level int) {
	var prefix string
	switch level {
	case LogDebug:
		prefix = "[DEBUG] "
		if !DEBUG {
			return
		}
	case LogInfo:
		prefix = "[INFO] "
	case LogWarning:
		prefix = "[WARNING] "
	case LogError:
		prefix = "[ERROR] "
	case LogSuccess:
		prefix = "[SUCCESS] "
	default:
		prefix = "[LOG] "
	}
	fmt.Println(prefix + message)
}
