package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	ytDlpExecutable = "yt-dlp"

	videoFormat = "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best" // Prioritize mp4 container
)

// downloadVideo uses yt-dlp to download the best quality video and audio.
// It returns the path to the downloaded video file.
func downloadVideo(ctx context.Context, logger *slog.Logger, url, workDir string) (string, error) {
	logger = logger.With("step", "downloadVideo", "url", url)
	logger.Info("Starting video download")

	// Ensure yt-dlp exists
	if err := checkExecutable(logger, ytDlpExecutable); err != nil {
		return "", err
	}

	// Define output template: videoID.ext (yt-dlp figures out the extension)
	// We need the video ID to predict the filename.
	videoID, err := getYoutubeVideoID(url)
	if err != nil {
		logger.Error("Failed to get video ID for filename prediction", "error", err)
		// Fallback or error out - let's error out for predictability
		return "", fmt.Errorf("cannot determine video ID from URL '%s' for download: %w", url, err)
	}
	outputTemplate := filepath.Join(workDir, fmt.Sprintf("%s.%%(ext)s", videoID)) // yt-dlp replaces %(ext)s

	// Build yt-dlp command arguments
	args := []string{
		"-f", videoFormat, // Select best mp4 video and audio, fallback to best overall
		"--merge-output-format", "mp4", // Ensure the final container is mp4
		"-o", outputTemplate, // Output filename template
		"--no-playlist", // Only download single video if URL is part of a playlist
		"--progress",    // Show progress
		"--no-warnings", // Suppress some common warnings
		// "--verbose",     // Uncomment for debugging yt-dlp issues
		url, // The video URL
	}

	// Execute the command
	if _, err := runCommand(ctx, logger, ytDlpExecutable, args...); err != nil {
		return "", fmt.Errorf("yt-dlp execution failed: %w", err)
	}

	// --- Predict the output filename ---
	// yt-dlp should have created a file like "videoID.mp4" or "videoID.webm" etc.
	// We need to find the exact name. Let's assume it's mp4 due to --merge-output-format mp4
	// A more robust way would be to parse yt-dlp's output if it reliably prints the filename,
	// or list the directory contents.
	expectedFilePath := filepath.Join(workDir, videoID+".mp4")

	// Basic check if the expected file exists
	if _, err := os.Stat(expectedFilePath); err != nil {
		logger.Warn("Expected video file not found directly, attempting to find it", "expectedPath", expectedFilePath)
		// Try finding *any* file starting with videoID in the workDir (less reliable)
		files, findErr := filepath.Glob(filepath.Join(workDir, videoID+".*"))
		if findErr != nil || len(files) == 0 {
			logger.Error("Could not find downloaded video file", "pattern", filepath.Join(workDir, videoID+".*"), "stat_error", err, "find_error", findErr)
			return "", fmt.Errorf("download command seemed successful, but couldn't locate output video file matching pattern %s.*", videoID)
		}
		// Filter out subtitle files etc. if necessary
		for _, f := range files {
			// Simple check for common video extensions
			ext := strings.ToLower(filepath.Ext(f))
			if ext == ".mp4" || ext == ".mkv" || ext == ".webm" || ext == ".avi" {
				expectedFilePath = f
				logger.Info("Found downloaded video file", "path", expectedFilePath)
				break
			}
		}
		// Re-check if we found a suitable file
		if _, err := os.Stat(expectedFilePath); err != nil {
			logger.Error("Still could not confirm downloaded video file path", "last_attempt", expectedFilePath)
			return "", fmt.Errorf("failed to confirm downloaded video file path after searching")
		}
	} else {
		logger.Info("Confirmed downloaded video file", "path", expectedFilePath)
	}

	return expectedFilePath, nil
}
