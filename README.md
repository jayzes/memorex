# Memorex

Convert video and audio files into Claude-friendly markdown with transcripts and keyframes.

Memorex extracts **transcripts** (via Whisper) and **keyframes** (via similarity detection) from video/audio files, outputting structured markdown that's ideal for analysis by Claude or other LLMs.

## Features

- **Smart keyframe extraction** - Uses normalized cross-correlation to detect significant visual changes
- **Accurate transcription** - Powered by whisper.cpp with timestamp segments
- **Auto-downloads models** - Whisper model downloads automatically on first use
- **Progress feedback** - Real-time progress bars for each processing step
- **Configurable output** - Adjust quality, scale, and similarity thresholds
- **Claude Code integration** - Includes a skill for automatic video analysis

## Demo

```
memorex

  Processing: demo.mp4
  Duration: 2m 34s

✓ Extracted 154 frames
✓ Found 12 keyframes
✓ Keyframes saved
✓ Audio extracted
✓ Transcribed 47 segments
✓ Markdown generated

✓ Output: demo_memorex.md
  Frames: demo_memorex_frames/
  Estimated tokens: ~15,600
```

## Installation

### Prerequisites

- **FFmpeg** - Required for video/audio processing
  ```bash
  # macOS
  brew install ffmpeg

  # Ubuntu/Debian
  sudo apt install ffmpeg
  ```

- **whisper.cpp** - Required for transcription
  ```bash
  # macOS
  brew install whisper-cpp

  # Ubuntu/Debian - build from source
  make install-whisper
  ```

### Install memorex

**Option 1: Go install (recommended)**
```bash
go install github.com/jayzes/memorex/cmd/memorex@latest
```

**Option 2: Build from source**
```bash
git clone https://github.com/jayzes/memorex.git
cd memorex
make build
# Binary is at ./bin/memorex
```

**Option 3: Full setup (includes whisper.cpp)**
```bash
git clone https://github.com/jayzes/memorex.git
cd memorex
make setup
make build
```

The Whisper model (~148MB) downloads automatically on first use to `~/.cache/whisper/`.

## Usage

```
memorex [options] <video-file>

Options:
  -o, --output      Output file path (default: <input>_memorex.md)
  -t, --threshold   Frame similarity threshold 0.0-1.0 (default: 0.85)
  -q, --quality     JPEG quality 1-100 (default: 30)
  -s, --scale       Frame scale factor (default: 0.5)
  -m, --model       Whisper model path (default: ~/.cache/whisper/ggml-base.bin)
  --no-transcript   Skip audio transcription
  --no-frames       Skip frame extraction (audio only)
```

### Examples

```bash
# Basic usage - outputs video_memorex.md + video_memorex_frames/
memorex video.mp4

# Custom output location
memorex -o analysis.md video.mp4

# Fewer keyframes (for fast-changing videos like presentations)
memorex -t 0.9 presentation.mov

# More keyframes (for static videos like interviews)
memorex -t 0.7 interview.mp4

# Audio only (podcasts, voice memos)
memorex --no-frames podcast.mp3

# Video only (skip transcription)
memorex --no-transcript screencast.mp4

# Smaller output files
memorex -q 20 -s 0.3 large_video.mp4
```

## Output Format

Memorex generates a markdown file alongside a frames directory:

```
video_memorex.md
video_memorex_frames/
├── frame_0001.jpg
├── frame_0015.jpg
└── frame_0089.jpg
```

The markdown contains:

```markdown
# Video Analysis: video.mp4

## Metadata
- Duration: 2m 34s
- Original frames: 154
- Keyframes extracted: 12
- Token estimate: ~15,600

## Transcript

[0:00] Welcome to this demonstration...
[0:15] As you can see on screen...

## Keyframes

### Frame 1 (0:00)
![Frame at 0:00](video_memorex_frames/frame_0001.jpg)

### Frame 15 (0:15)
![Frame at 0:15](video_memorex_frames/frame_0015.jpg)
```

## How It Works

1. **Frame extraction** - Extracts frames at 1 fps using FFmpeg
2. **Keyframe detection** - Compares consecutive frames using normalized cross-correlation; frames below the similarity threshold are marked as keyframes
3. **Audio transcription** - Extracts audio to 16kHz mono WAV, transcribes with whisper.cpp
4. **Output generation** - Combines transcript and keyframes into structured markdown

## Claude Code Integration

Memorex includes a Claude Code skill for automatic video analysis. To install:

```
/install-plugin github.com/jayzes/memorex
```

Or simply ask Claude Code:

> Install the memorex plugin from github.com/jayzes/memorex

Once installed, Claude will automatically use memorex when you ask to analyze videos.

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Setup git hooks (runs lint + test on commit)
make setup-hooks
```

## Troubleshooting

**FFmpeg not found**
```bash
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg
```

**whisper-cli not found**
```bash
# macOS
brew install whisper-cpp

# Or build from source
make install-whisper
# Then add to PATH or symlink:
ln -sf ~/.local/share/whisper.cpp/src/build/bin/whisper-cli /usr/local/bin/whisper-cli
```

**Build errors with whisper.cpp**
```bash
# Ensure CMake and C++ compiler are installed
# macOS
brew install cmake && xcode-select --install

# Ubuntu/Debian
sudo apt install cmake build-essential
```

**Out of memory with large videos**
```bash
# Reduce frame scale and increase threshold
memorex -s 0.25 -t 0.95 large_video.mp4
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`make test lint`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.
