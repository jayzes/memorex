![Blown away](header.webp)

*(Yes, I know this is technically a Maxell ad, not a Memorex one. But it's a better image.)*

# Memorex

*Is it live, or is it Memorex?*

Give Claude perfect recall of any video. Memorex converts video and audio into structured markdown with transcripts and keyframes—so Claude can see what you saw and hear what you heard.

## Why Memorex?

Claude can't watch videos. But it can read markdown and view images. Memorex bridges that gap:

- **Transcripts with timestamps** — Every word, synced to the timeline
- **Smart keyframes** — Only the frames that matter, not 30 fps of redundancy
- **One command** — Point it at a video, get back something Claude understands

## Demo

```
$ memorex demo.mp4

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

## Install

**Prerequisites:** FFmpeg and whisper.cpp

```bash
# macOS
brew install ffmpeg whisper-cpp

# Ubuntu/Debian
sudo apt install ffmpeg
make install-whisper  # builds whisper.cpp from source
```

**Install memorex:**

```bash
go install github.com/jayzes/memorex/cmd/memorex@latest
```

The Whisper model (~148MB) downloads automatically on first run.

## Usage

```bash
memorex video.mp4                    # Basic usage
memorex -t 0.9 presentation.mov      # Fewer keyframes (fast-changing video)
memorex -t 0.7 interview.mp4         # More keyframes (static video)
memorex --no-frames podcast.mp3      # Audio only
memorex --no-transcript silent.mp4   # Video only
memorex -q 20 -s 0.3 huge.mp4        # Smaller output
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | `<input>_memorex.md` | Output path |
| `-t, --threshold` | `0.85` | Frame similarity (lower = more keyframes) |
| `-q, --quality` | `30` | JPEG quality (1-100) |
| `-s, --scale` | `0.5` | Frame scale factor |
| `--no-transcript` | | Skip transcription |
| `--no-frames` | | Skip frame extraction |

## Output

```
video_memorex.md
video_memorex_frames/
├── frame_0001.jpg
├── frame_0015.jpg
└── frame_0089.jpg
```

The markdown gives Claude everything it needs:

```markdown
# Video Analysis: video.mp4

## Metadata
- Duration: 2m 34s
- Keyframes: 12
- Token estimate: ~15,600

## Transcript

[0:00] Welcome to this demonstration...
[0:15] As you can see on screen...

## Keyframes

### Frame 1 (0:00)
![Frame at 0:00](video_memorex_frames/frame_0001.jpg)
```

## Claude Code Plugin

Let Claude handle everything automatically.

```
/plugin marketplace add jayzes/memorex
/plugin install memorex@jayzes-memorex
```

Then just ask Claude to analyze a video. It'll run memorex, read the output, and tell you what it sees.

## How It Works

1. **Extract** — FFmpeg pulls frames at 1 fps
2. **Compare** — Normalized cross-correlation finds visually distinct frames
3. **Transcribe** — whisper.cpp converts speech to timestamped text
4. **Package** — Everything becomes Claude-readable markdown

## Development

```bash
make test    # Run tests
make lint    # Run linter
make fmt     # Format code
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| FFmpeg not found | `brew install ffmpeg` or `apt install ffmpeg` |
| whisper-cli not found | `brew install whisper-cpp` or `make install-whisper` |
| Out of memory | Use `-s 0.25 -t 0.95` for large videos |

## Contributing

Fork, branch, code, test, PR. The usual.

## License

MIT
