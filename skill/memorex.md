# Skill: Analyze Video

Process video files with memorex to extract transcripts and keyframes for analysis.

## Triggers

- User asks to analyze, watch, or review a video file
- User provides a screen recording or demo video
- User wants to understand what happens in a video
- User mentions .mp4, .mov, .webm, or other video formats
- User says "memorex" or asks to run memorex

## Installation

### Prerequisites

1. **FFmpeg** - Required for video/audio processing
   ```bash
   # macOS
   brew install ffmpeg

   # Ubuntu/Debian
   sudo apt install ffmpeg

   # Verify installation
   ffmpeg -version
   ```

2. **whisper.cpp** - Required for transcription
   ```bash
   # macOS
   brew install whisper-cpp

   # Or build from source
   git clone https://github.com/ggerganov/whisper.cpp
   cd whisper.cpp
   make
   sudo cp main /usr/local/bin/whisper-cli
   ```

   Note: The Whisper model (~148MB) will be automatically downloaded on first use.

### Install memorex

**From source (requires Go 1.21+):**
```bash
# Clone and build
git clone https://github.com/jayzes/memorex
cd memorex
go build -o memorex ./cmd/memorex

# Move to PATH
sudo mv memorex /usr/local/bin/

# Or install directly
go install github.com/jayzes/memorex/cmd/memorex@latest
```

**Verify installation:**
```bash
memorex --help
```

## Workflow

1. **Confirm the video file exists**
   - Verify the file path is accessible
   - Note the file size and format

2. **Run memorex**
   ```bash
   memorex -o /tmp/video_analysis.md <video-path>
   ```

   Options to consider:
   - `-t 0.9` for fewer keyframes (less similar frames filtered)
   - `-t 0.7` for more keyframes (more sensitive to changes)
   - `--no-transcript` if only visual analysis needed
   - `--no-frames` for audio-only analysis

3. **Report extraction results**
   - Number of keyframes extracted
   - Transcript length
   - Estimated token cost

4. **Ask user what to analyze**
   - Full video walkthrough?
   - Specific UI elements?
   - Looking for bugs or issues?
   - Compare to expected behavior?

5. **Read and analyze the output**
   - Load the markdown file
   - Read keyframe images as needed
   - Provide analysis based on user's goal

## Cost Optimization

For large videos (>30 keyframes), suggest:
- Increase threshold (`-t 0.9`) to extract fewer frames
- Focus on specific time ranges if the user knows where to look
- Start with transcript-only analysis to identify relevant sections

## Example Commands

```bash
# Standard analysis
memorex video.mp4

# High-change video (presentations, demos)
memorex -t 0.9 demo.mov

# Static video (talking head, minimal visual changes)
memorex -t 0.7 interview.mp4

# Audio-only (podcast, voice memo)
memorex --no-frames podcast.mp3

# Custom output location
memorex -o ~/analysis/meeting.md recording.mp4

# Lower quality frames (smaller files)
memorex -q 20 -s 0.3 large_video.mp4
```

## Troubleshooting

**memorex not found:**
- Ensure it's in your PATH: `which memorex`
- Try reinstalling: `go install github.com/jayzes/memorex/cmd/memorex@latest`

**FFmpeg errors:**
- Check FFmpeg is installed: `which ffmpeg`
- Update FFmpeg: `brew upgrade ffmpeg` (macOS)

**Transcription fails:**
- Check whisper-cli is installed: `which whisper-cli`
- The model auto-downloads to `~/.cache/whisper/ggml-base.bin`
- Try `--no-transcript` to test video extraction separately

**Memory issues with large videos:**
- Reduce frame scale: `-s 0.25`
- Increase threshold for fewer frames: `-t 0.95`
- Process in segments if needed
