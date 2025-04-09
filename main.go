package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	// --- Configuration ---
	videoURL := flag.String("url", "", "YouTube video URL (required)")
	outputDir := flag.String("output", "./output", "Directory for final processed video")
	keepWorkDir := flag.Bool("keep-workdir", true, "Keep the temporary working directory after processing")
	logLevelStr := flag.String("log-level", os.Getenv("LOG_LEVEL"), "Log level (DEBUG, INFO, WARN, ERROR). Overrides LOG_LEVEL env var.")
	flag.Parse()

	if *videoURL == "" {
		fmt.Println("Error: -url flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// --- Logging Setup ---
	var logLevel slog.Level
	switch *logLevelStr {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(logHandler).With("service", "youtube-processor")
	slog.SetDefault(logger) // Set as default for convenience

	logger.Info("Starting YouTube processing pipeline",
		"url", *videoURL,
		"outputDir", *outputDir,
		"keepWorkDir", *keepWorkDir,
		"logLevel", logLevel.String(),
	)

	// --- Main Processing Logic ---
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour) // Add a global timeout
	defer cancel()

	finalPath, err := processVideoPipeline(ctx, logger, *videoURL, *outputDir, *keepWorkDir)
	if err != nil {
		logger.Error("Video processing pipeline failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Pipeline completed successfully!", "finalVideoPath", finalPath)
}

// processVideoPipeline orchestrates the entire workflow.
func processVideoPipeline(ctx context.Context, logger *slog.Logger, videoURL, outputBaseDir string, keepWorkDir bool) (string, error) {
	startTime := time.Now()

	// 1. Get Video ID and create working directory
	videoID, err := getYoutubeVideoID(logger, videoURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract video ID: %w", err)
	}

	logger = logger.With("videoID", videoID) // Add videoID to all subsequent logs
	logger.Info("Extracted video ID")

	// Create a base directory for all processing artifacts if it doesn't exist
	if err := ensureDir(logger, outputBaseDir); err != nil {
		return "", fmt.Errorf("failed to ensure base output directory %s: %w", outputBaseDir, err)
	}

	// Create a unique working directory for this specific video inside the base output dir
	workDir, err := getWorkDir(outputBaseDir, videoID)
	if err != nil {
		return "", fmt.Errorf("failed to create working directory: %w", err)
	}
	logger.Info("Created working directory", "path", workDir)

	// Defer cleanup of the working directory unless requested otherwise
	if !keepWorkDir {
		defer func() {
			logger.Info("Cleaning up working directory", "path", workDir)
			if rmErr := os.RemoveAll(workDir); rmErr != nil {
				logger.Error("Failed to remove working directory", "path", workDir, "error", rmErr)
			}
		}()
	} else {
		logger.Info("Keeping working directory as requested", "path", workDir)
	}

	// 2. Download Video
	videoPath, err := downloadVideo(ctx, logger, videoID, videoURL, workDir)
	if err != nil {
		return "", fmt.Errorf("step 1: download video failed: %w", err)
	}

	// 3. Download Subtitles
	srtPath, err := downloadSubtitles(ctx, logger, videoID, workDir)
	if err != nil {
		// Consider if this should be a fatal error. Maybe the user wants the video even without subs?
		// For this flow, we assume subtitles are required.
		return "", fmt.Errorf("step 2: download subtitles failed: %w", err)
	}

	// 4. Merge Video and Subtitles
	// Place the final merged file directly into the user-specified outputBaseDir
	finalVideoPath, err := mergeVideoSubtitles(ctx, logger, videoPath, srtPath, outputBaseDir, videoID)
	if err != nil {
		return "", fmt.Errorf("step 4: merge video and subtitles failed: %w", err)
	}

	// If we are keeping the work directory, the original downloaded video is still there.
	// If we are *not* keeping the work directory, the original video download will be deleted by the deferred cleanup.
	// The final output is placed *outside* the workDir (in outputBaseDir), so it's always preserved.

	logger.Info("Processing finished", "totalDuration", time.Since(startTime))
	return finalVideoPath, nil
}
