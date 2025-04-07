package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	ffmpegExecutable = "ffmpeg"
)

// mergeVideoSubtitles uses ffmpeg to merge the video and translated subtitle file.
func mergeVideoSubtitles(ctx context.Context, logger *slog.Logger, videoPath, subtitlePath, outputDir, videoID string) (string, error) {
	logger = logger.With("step", "mergeVideoSubtitles", "video", videoPath, "subtitles", subtitlePath)
	logger.Info("Starting video and subtitle merge")

	// Ensure ffmpeg exists
	if err := checkExecutable(logger, ffmpegExecutable); err != nil {
		return "", err
	}

	// Define the final output path
	// Ensure output directory exists (should be created by getWorkDir or main)
	finalFileName := fmt.Sprintf("%s_final_with_%s_subs.mp4", videoID, targetLang)
	finalOutputPath := filepath.Join(outputDir, finalFileName) // Place final output directly in the specified output dir

	// Build ffmpeg command arguments
	args := []string{
		"-i", videoPath,
		"-c:a", "copy", // Copy audio stream without re-encoding
		"-vf", fmt.Sprintf("subtitles=%s:force_style='FontSize=16,Alignment=2'", subtitlePath), // Burn subtitles into video
		"-y", // Overwrite output
		finalOutputPath,
	}

	// Check if the subtitle file is empty. If so, don't add subtitle arguments.
	subFileInfo, err := os.Stat(subtitlePath)
	isEmptySubtitle := err == nil && subFileInfo.Size() == 0

	if isEmptySubtitle {
		logger.Warn("Subtitle file is empty, copying video without subtitles")
		// Simplified args without subtitle if SRT is empty
		args = []string{
			"-i", videoPath,
			"-c", "copy", // Copy existing streams without re-encoding
			"-y",
			finalOutputPath,
		}
	}

	// Execute the command
	if _, err := runCommand(ctx, logger, ffmpegExecutable, args...); err != nil {
		// Attempt to remove potentially incomplete output file on error
		_ = os.Remove(finalOutputPath)
		return "", fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	// Verify the final output file exists
	if _, err := os.Stat(finalOutputPath); err != nil {
		logger.Error("ffmpeg command seemed successful, but final output file not found", "path", finalOutputPath, "error", err)
		return "", fmt.Errorf("ffmpeg finished, but expected output file '%s' was not found: %w", finalOutputPath, err)
	}

	logger.Info("Video and subtitles merged successfully", "path", finalOutputPath)
	return finalOutputPath, nil
}
