package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runCommand executes an external command and logs its output.
// It returns an error if the command fails to start or exit with a non-zero status.
func runCommand(ctx context.Context, logger *slog.Logger, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	logger = logger.With("command", name, "args", strings.Join(args, " ")) // Add command context to logger
	logger.Info("Executing command")

	startTime := time.Now()
	output, err := cmd.CombinedOutput() // Capture both stdout and stderr
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Command execution failed", "duration", duration, "error", err, "output", string(output))
		// %w wraps the original error for better debugging
		return nil, fmt.Errorf("command '%s %s' failed: %w\nOutput: %s", name, strings.Join(args, " "), err, string(output))
	}

	logger.Info("Command executed successfully", "duration", duration)
	return output, nil
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(logger *slog.Logger, dirPath string) error {
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		logger.Error("Failed to create directory", "path", dirPath, "error", err)
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}
	logger.Debug("Ensured directory exists", "path", dirPath)
	return nil
}

// checkExecutable verifies if an executable exists in the system's PATH.
func checkExecutable(logger *slog.Logger, name string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		logger.Error("Required executable not found in PATH", "executable", name, "error", err)
		return fmt.Errorf("executable '%s' not found in PATH: %w. Please ensure it is installed and accessible", name, err)
	}
	logger.Debug("Executable found", "name", name, "path", path)
	return nil
}

// getYoutubeVideoID extracts the video ID from various YouTube URL formats.
func getYoutubeVideoID(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL '%s': %w", rawURL, err)
	}

	// Standard youtube.com format (youtube.com/watch?v=VIDEO_ID)
	if strings.Contains(parsedURL.Host, "youtube.com") {
		videoID := parsedURL.Query().Get("v")
		if videoID != "" {
			return videoID, nil
		}
	}

	// Shortened youtu.be format (youtu.be/VIDEO_ID)
	if strings.Contains(parsedURL.Host, "youtu.be") {
		videoID := strings.TrimPrefix(parsedURL.Path, "/")
		if videoID != "" {
			// Remove potential query params like ?t=...
			if idx := strings.Index(videoID, "?"); idx != -1 {
				videoID = videoID[:idx]
			}
			return videoID, nil
		}
	}

	// Handle other potential formats or return error
	return "", fmt.Errorf("could not extract video ID from URL: %s", rawURL)
}

// getWorkDir creates a unique working directory for processing a video.
func getWorkDir(baseDir, videoID string) (string, error) {
	// Sanitize videoID for use in directory name if necessary, though usually safe.
	// Use timestamp for uniqueness in case of reruns, though videoID is often unique enough.
	// For simplicity, just use videoID for now.
	workDir := filepath.Join(baseDir, "processing_"+videoID)
	if err := os.MkdirAll(workDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create working directory %s: %w", workDir, err)
	}
	return workDir, nil
}
