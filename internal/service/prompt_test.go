package service

import (
	"strings"
	"testing"
)

func TestBuildPromptOptimizePromptFillsUserInput(t *testing.T) {
	userInput := "Create a landing page for a fitness course"

	prompt := buildPromptOptimizePrompt(userInput)

	if strings.Contains(prompt, "{{user_input}}") {
		t.Fatal("prompt still contains user_input placeholder")
	}
	if !strings.Contains(prompt, userInput) {
		t.Fatalf("prompt does not contain user input: %q", prompt)
	}
}

func TestNormalizePromptSceneDefaults(t *testing.T) {
	if got := normalizePromptScene("  "); got != "default" {
		t.Fatalf("normalizePromptScene() = %q, want default", got)
	}
}

func TestCleanOptimizedPromptRemovesLabelAndQuotes(t *testing.T) {
	input := "Optimized Prompt:\n\n\"Close-up photograph of a soft, textured throw pillow.\""

	got := cleanOptimizedPrompt(input)

	want := "Close-up photograph of a soft, textured throw pillow."
	if got != want {
		t.Fatalf("cleanOptimizedPrompt() = %q, want %q", got, want)
	}
}

func TestCleanOptimizedPromptRemovesMarkdownLabel(t *testing.T) {
	input := "**Optimized Prompt:**\n\nCreate a serene product photograph."

	got := cleanOptimizedPrompt(input)

	want := "Create a serene product photograph."
	if got != want {
		t.Fatalf("cleanOptimizedPrompt() = %q, want %q", got, want)
	}
}
