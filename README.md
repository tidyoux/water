# Water

Water 是一个视频处理工具，主要用于下载视频并进行字幕处理。

## 功能特点

- 使用 yt-dlp 下载 youtube 最佳质量视频
- 使用 youtube_transcript_api 下载视频英文字幕
- 使用 openai api 将字幕翻译成中文
- 使用 ffmpeg 将视频和字幕合并

## 环境变量配置

使用前需要配置以下环境变量：

```bash
OPENAI_API_KEY=your_api_key
OPENAI_BASE_URL=your_base_url    # 可选
OPENAI_MODEL=your_model          # 可选，默认为 gpt-4o-mini
```

## 安装

```bash
git clone [repository_url]
cd water
go mod download
```

## 使用方法

```bash
go run . [flags]
```

### 可用的命令行参数

- `-url`: 指定视频 URL
- `-output`: 指定输出目录
- `-keep-workdir`: 保留工作目录

## 示例

```bash
go run . -output ./output -url https://www.youtube.com/watch?v=example
```

## 项目结构

- `main.go`: 主程序入口
- `video_downloader.go`: 视频下载模块
- `subtitle_handler.go`: 字幕处理模块
- `merger.go`: 视频合并模块
- `util.go`: 工具函数

## 许可证

MIT License

## 贡献

欢迎提交 Issues 和 Pull Requests！
