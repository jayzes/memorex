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

**IMPORTANT: Run this entire workflow as a subagent** using the `Agent` tool with `subagent_type: "general-purpose"`. This keeps the potentially large memorex output (transcripts, keyframe images) out of the main conversation context.

### Subagent prompt template

Launch a single `general-purpose` Agent with a prompt like the following (fill in the bracketed values):

```
Process and analyze a video file for the user.

Video path: [video-path]
User's question/goal: [what the user wants to know about the video]

Follow these steps:

1. Confirm the video file exists (ls the path, note file size and format).

2. Run memorex:
   ```bash
   mkdir -p /tmp/memorex
   memorex -o /tmp/memorex/[video-basename]_analysis.md [video-path]
   ```
   Options to consider:
   - `-t 0.9` for fewer keyframes (less similar frames filtered)
   - `-t 0.7` for more keyframes (more sensitive to changes)
   - `--no-transcript` if only visual analysis needed
   - `--no-frames` for audio-only analysis

3. Read the generated markdown file at `/tmp/memorex/[video-basename]_analysis.md` using the Read tool.

4. Review the metadata (duration, frame count, keyframe count, token estimate).

5. Read relevant keyframe images from the frames directory using the Read tool (Claude can see images). Cross-reference transcript timestamps with keyframe timestamps.

6. Based on the user's goal, provide a thorough analysis covering:
   - Summary of what happens in the video
   - Relevant details tied to the user's question
   - Key moments with timestamps
   - Descriptions of what's visible in important keyframes

Return a concise but complete analysis. Include the output file path so the user can reference it later.
```

### What to do in the main conversation

1. Confirm the video file path with the user if ambiguous
2. Launch the Agent subagent with the prompt above
3. Relay the agent's analysis back to the user in a concise summary
4. If the user has follow-up questions, launch another Agent to re-read the memorex output files and answer specifically

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

## Follow-up Questions

If the user has follow-up questions about a previously analyzed video, launch another `general-purpose` Agent subagent with a prompt that tells it to re-read the memorex output files at `/tmp/memorex/` and answer the specific question. This avoids loading the full output into the main context.

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
