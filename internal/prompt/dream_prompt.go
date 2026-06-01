package prompt

import (
	"fmt"
	"os"
)

// GenerateDreamPromptFile generates the dream phase prompt and saves it to a temporary file.
// The dream phase guides skill evolution and system improvement analysis.
// Returns the file path; caller is responsible for cleanup.
func GenerateDreamPromptFile(requirement string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	content := "# Rick Dream Phase\n\n"
	if requirement != "" {
		content += "## Requirement\n\n" + requirement + "\n\n"
	}

	// Inject core skills for dream phase
	coreSkills := LoadCoreSkills([]string{"sense", "evolve-skills"})
	if coreSkills != "" {
		content += "## Core Skills\n\n" + coreSkills + "\n"
	}

	tmpFile, err := os.CreateTemp("", "rick-dream-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write dream prompt: %w", err)
	}

	return tmpFile.Name(), nil
}
