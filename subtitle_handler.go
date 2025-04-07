package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	youtubeTranscriptApiExecutable = "youtube_transcript_api"
	sourceLang                     = "en" // Language to download
	targetLang                     = "zh" // Language to translate
	targetFormat                   = "srt"
)

// downloadSubtitles calls the Python script to download subtitles using youtube_transcript_api.
// It returns the path to the downloaded SRT file.
func downloadSubtitles(ctx context.Context, logger *slog.Logger, videoID, workDir string) (string, error) {
	logger = logger.With("step", "downloadSubtitles", "videoID", videoID)
	logger.Info("Starting subtitle download")

	// Ensure Python exists
	if err := checkExecutable(logger, youtubeTranscriptApiExecutable); err != nil {
		return "", err
	}

	// Define output path for the original English SRT file
	originalSrtPath := filepath.Join(workDir, fmt.Sprintf("%s_%s.srt", videoID, sourceLang))

	args := []string{
		"--languages", sourceLang,
		"--format", targetFormat,
		videoID,
	}

	// Execute the command
	if output, err := runCommand(ctx, logger, youtubeTranscriptApiExecutable, args...); err != nil {
		return "", fmt.Errorf("youtube_transcript_api execution failed: %w", err)
	} else {
		// Write the output to the original SRT file
		if err := os.WriteFile(originalSrtPath, output, 0644); err != nil {
			return "", fmt.Errorf("failed to write SRT file %s: %w", originalSrtPath, err)
		}
	}

	// Verify the output file was created
	if _, err := os.Stat(originalSrtPath); err != nil {
		logger.Error("Subtitle script ran but output SRT file not found", "path", originalSrtPath, "error", err)
		return "", fmt.Errorf("subtitle script finished, but expected SRT file '%s' was not created: %w", originalSrtPath, err)
	}

	logger.Info("Subtitles downloaded successfully", "path", originalSrtPath)
	return originalSrtPath, nil
}

// translateSubtitles reads an SRT file, sends its content to OpenAI for translation,
// and saves the translated content to a new SRT file.
func translateSubtitles(ctx context.Context, logger *slog.Logger, openaiClient *openai.Client, openAIModel, videoID, originalSrtPath, workDir string) (string, error) {
	logger = logger.With("step", "translateSubtitles", "sourceSrt", originalSrtPath)
	logger.Info("Starting subtitle translation")

	// Read the original SRT content
	srtContentBytes, err := os.ReadFile(originalSrtPath)
	if err != nil {
		logger.Error("Failed to read original SRT file", "error", err)
		return "", fmt.Errorf("failed to read SRT file %s: %w", originalSrtPath, err)
	}
	srtContent := string(srtContentBytes)

	if len(strings.TrimSpace(srtContent)) == 0 {
		logger.Warn("Original SRT file is empty, skipping translation")
		// Return an empty path or handle as needed - maybe create empty translated file?
		// Let's create an empty file for consistency in the merge step.
		translatedSrtPath := filepath.Join(workDir, fmt.Sprintf("%s_%s_translated.srt", videoID, targetLang))
		if err := os.WriteFile(translatedSrtPath, []byte{}, 0644); err != nil {
			logger.Error("Failed to write empty translated SRT file", "path", translatedSrtPath, "error", err)
			return "", fmt.Errorf("failed to write empty translated SRT file %s: %w", translatedSrtPath, err)
		}
		logger.Info("Created empty translated SRT file as original was empty", "path", translatedSrtPath)
		return translatedSrtPath, nil // Return path to empty file
	}

	// Prepare the prompt for OpenAI
	// Using ChatCompletion is generally recommended over legacy Completion
	systemPrompt := fmt.Sprintf("You are a professional translator specialized in video subtitles. Translate the following English subtitles into %s. Maintain the original SRT format, including timestamps and sequence numbers, exactly. Only output the translated SRT content.", targetLang)
	userPrompt := srtContent

	// Call OpenAI API
	logger.Info("Sending translation request to OpenAI", "model", openAIModel) // Or specify another model
	startTime := time.Now()
	resp, err := openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openAIModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.2, // Lower temperature for more deterministic translation
		},
	)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("OpenAI API call failed", "duration", duration, "error", err)
		return "", fmt.Errorf("openai API request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		logger.Error("OpenAI response was empty or invalid", "duration", duration, "response", resp)
		return "", fmt.Errorf("openai returned an empty or invalid response")
	}

	translatedContent := resp.Choices[0].Message.Content
	logger.Info("Received translation from OpenAI", "duration", duration, "usage", resp.Usage) // Log token usage

	// Define path for the translated SRT file
	translatedSrtPath := filepath.Join(workDir, fmt.Sprintf("%s_%s_translated.srt", videoID, targetLang))

	// Write the translated content to the new file
	err = os.WriteFile(translatedSrtPath, []byte(translatedContent), 0644) // Sensible default permissions
	if err != nil {
		logger.Error("Failed to write translated SRT file", "path", translatedSrtPath, "error", err)
		return "", fmt.Errorf("failed to write translated SRT file %s: %w", translatedSrtPath, err)
	}

	logger.Info("Subtitles translated successfully", "path", translatedSrtPath)
	return translatedSrtPath, nil
}
