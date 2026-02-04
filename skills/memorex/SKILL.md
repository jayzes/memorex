---
name: analyze-video
description: Process video files with memorex to extract transcripts and keyframes for analysis
---

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

   Memorex generates two outputs:

   **Markdown file** (`<input>_memorex.md`):
   - Contains metadata (duration, frame count, token estimate)
   - Full transcript with timestamps in `[M:SS]` format
   - Keyframe references with timestamps
   - Use the `Read` tool to load this file

   **Frames directory** (`<input>_memorex_frames/`):
   - Contains JPEG images of each keyframe
   - Named `frame_NNNN.jpg` where NNNN is the frame number
   - Use the `Read` tool to view these images - Claude can see image contents
   - Keyframes are ordered chronologically and correspond to transcript timestamps

   **Analysis approach**:
   - First read the markdown file to understand the video structure
   - Read specific keyframe images when you need to see what's on screen at a particular moment
   - Cross-reference transcript timestamps with keyframe timestamps to correlate audio and visuals
   - The token estimate helps gauge how much of the output can be processed in one context

## Output Format

Memorex generates a structured markdown file with this format:

```markdown
# Video Analysis: example.mp4

## Metadata
- Duration: 2m 34s
- Original frames: 154
- Keyframes extracted: 12
- Token estimate: ~15,600

## Transcript

[0:00] First spoken words...
[0:15] More dialogue here...
[1:30] Later in the video...

## Keyframes

### Frame 1 (0:00)
![Frame at 0:00](example_memorex_frames/frame_0001.jpg)

### Frame 15 (0:15)
![Frame at 0:15](example_memorex_frames/frame_0015.jpg)
```

**Interpreting the output:**
- Timestamps in transcript (`[M:SS]`) indicate when words were spoken
- Keyframes are captured at moments of significant visual change
- Frame numbers correspond to seconds into the video (at 1fps extraction)
- To see what was on screen when something was said, find the keyframe with the closest timestamp

## Cost Optimization

For large videos (>30 keyframes), suggest:
- Increase threshold (`-t 0.9`) to extract fewer frames
- Focus on specific time ranges if the user knows where to look
- Start with transcript-only analysis to identify relevant sections

## After Running Memorex

Once memorex completes, follow these steps to analyze the video:

1. **Read the markdown file** using the `Read` tool:
   ```
   Read the file at /path/to/video_memorex.md
   ```

2. **Review the metadata** to understand scope:
   - Duration tells you video length
   - Token estimate helps plan analysis depth
   - Keyframe count indicates visual complexity

3. **Scan the transcript** for relevant sections:
   - Look for keywords related to user's question
   - Note timestamps of interesting moments

4. **View specific keyframes** as needed:
   ```
   Read the image at /path/to/video_memorex_frames/frame_0015.jpg
   ```
   - View frames that correspond to important transcript moments
   - Compare consecutive keyframes to understand transitions

5. **Correlate audio and visuals**:
   - Match transcript timestamps to nearest keyframes
   - Describe what's being shown while specific things are being said

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
