package gojudge

// Config contains language-specific configurations for compilation and execution
type LanguageConfig struct {
	Image   string   // Not used strictly in go-judge sandbox but good for reference
	Compile *RunCmd  // Command to compile the code (if needed)
	Run     RunCmd   // Command to run the code
}

type RunCmd struct {
	Command []string
	Env     []string
}

// GetLanguageConfig returns the compilation and execution config for a given language.
// These commands assume the code is written to a file based on the language.
func GetLanguageConfig(language string, sourceFile string, exeFile string) (*LanguageConfig, bool) {
	configs := map[string]LanguageConfig{
		"CPP": {
			Compile: &RunCmd{
				Command: []string{"/usr/bin/g++", "-O3", "-std=c++17", sourceFile, "-o", exeFile},
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
			Run: RunCmd{
				Command: []string{exeFile},
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
		},
		"GO": {
			Compile: &RunCmd{
				Command: []string{"/usr/local/go/bin/go", "build", "-o", exeFile, sourceFile},
				Env:     []string{"PATH=/usr/local/go/bin:/usr/bin:/bin", "GOCACHE=/tmp", "GOPATH=/tmp/go", "CGO_ENABLED=0", "GOPROXY=off", "GOSUMDB=off"},
			},
			Run: RunCmd{
				Command: []string{exeFile},
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
		},
		"PYTHON": {
			Compile: nil, // Python is interpreted
			Run: RunCmd{
				Command: []string{"/usr/bin/python3", sourceFile},
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
		},
		"JAVA": {
			Compile: &RunCmd{
				Command: []string{"/usr/bin/javac", sourceFile}, // Generates .class
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
			Run: RunCmd{
				Command: []string{"/usr/bin/java", "-cp", ".", "Main"},
				Env:     []string{"PATH=/usr/bin:/bin"},
			},
		},
	}

	cfg, ok := configs[language]
	return &cfg, ok
}

// GetSourceFileName returns the default source file name for a given language.
func GetSourceFileName(language string) string {
	switch language {
	case "CPP":
		return "main.cpp"
	case "GO":
		return "main.go"
	case "PYTHON":
		return "main.py"
	case "JAVA":
		return "Main.java"
	default:
		return "main.txt"
	}
}

// GetExeFileName returns the default executable file name for a given language.
func GetExeFileName(language string) string {
	switch language {
	case "CPP":
		return "main"
	case "GO":
		return "main"
	case "JAVA":
		return "Main.class"
	default:
		return "main.out"
	}
}
