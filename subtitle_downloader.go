package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	youtubeTranscriptApiExecutable = "youtube_transcript_api"
	targetLang                     = "zh-Hans" // Language to translate
	targetFormat                   = "srt"
)

// downloadSubtitles uses youtube_transcript_api to download subtitle
func downloadSubtitles(ctx context.Context, logger *slog.Logger, videoID, workDir string) (string, error) {
	logger = logger.With("step", "downloadSubtitles", "videoID", videoID)
	logger.Info("Starting subtitle download")

	if err := checkExecutable(logger, youtubeTranscriptApiExecutable); err != nil {
		return "", err
	}

	srtPath := filepath.Join(workDir, fmt.Sprintf("%s_%s.srt", videoID, targetLang))

	args := []string{
		"--translate", targetLang,
		"--format", targetFormat,
		videoID,
	}

	if output, err := runCommand(ctx, logger, youtubeTranscriptApiExecutable, args...); err != nil {
		return "", fmt.Errorf("youtube_transcript_api execution failed: %w", err)
	} else {
		if len(output) == 0 {
			return "", fmt.Errorf("youtube_transcript_api returned empty output")
		}

		// Write the output to the original SRT file
		if err := os.WriteFile(srtPath, output, 0644); err != nil {
			return "", fmt.Errorf("failed to write SRT file %s: %w", srtPath, err)
		}
	}

	// Verify the output file was created
	if _, err := os.Stat(srtPath); err != nil {
		logger.Error("Subtitle script ran but output SRT file not found", "path", srtPath, "error", err)
		return "", fmt.Errorf("subtitle script finished, but expected SRT file '%s' was not created: %w", srtPath, err)
	}

	logger.Info("Subtitles downloaded successfully", "path", srtPath)
	return srtPath, nil
}
