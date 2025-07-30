# Water

Water 是一个视频处理工具，主要用于下载视频并进行字幕处理。

## 功能特点

- 使用 yt-dlp 下载 youtube 最佳质量视频和中文字幕
- 使用 ffmpeg 将视频和字幕合并

## 快速开始

```bash
git clone https://github.com/tidyoux/water.git
cd water
go mod tidy
go build -o water
```

## 示例

```bash
./water -output ./output -url https://www.youtube.com/watch?v=example
```

## 可用的命令行参数

```
Usage of water:
  -keep-workdir
        Keep the temporary working directory after processing (default true)
  -log-level string
        Log level (DEBUG, INFO, WARN, ERROR). Overrides LOG_LEVEL env var.
  -output string
        Directory for final processed video (default "./output")
  -url string
        YouTube video URL (required)
```
