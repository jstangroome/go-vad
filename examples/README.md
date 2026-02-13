# Examples

This directory contains example programs demonstrating how to use the go-vad package.

## Prerequisites

- Go 1.16 or later
- For non-WAV files: ffmpeg installed on your system

## Running the Examples

### 1. Basic VAD Example

Demonstrates simple usage of the Basic VAD with fixed thresholds.

```bash
cd examples/basic
go run main.go path/to/audio.wav
```

**Output**: Lists all detected speech segments with timing and energy information.

### 2. Adaptive VAD Example

Shows how to use the Adaptive VAD which automatically adjusts to audio characteristics.

```bash
cd examples/adaptive
go run main.go path/to/audio.wav
```

**Output**: Displays detected segments with adaptive thresholding, includes summary statistics.

### 3. Comparison Example

Compares Basic VAD vs Adaptive VAD on the same audio file.

```bash
cd examples/comparison
go run main.go path/to/audio.wav
```

**Output**: Side-by-side comparison of both methods with performance metrics.

## Building the Examples

You can build standalone executables:

```bash
# Basic example
cd examples/basic
go build -o vad-basic

# Adaptive example
cd examples/adaptive
go build -o vad-adaptive

# Comparison example
cd examples/comparison
go build -o vad-compare
```

## Sample Output

```
Loading audio file: conversation.wav
Audio loaded successfully:
  File: conversation.wav
  Duration: 15.234s
  Sample Rate: 16000 Hz
  Channels: 1

Running Adaptive VAD...

Detected 5 speech segments:
---
Segment 1:
  Start:    0.320s
  End:      3.450s
  Duration: 3.130s
  Energy:   0.0245

Segment 2:
  Start:    4.120s
  End:      7.890s
  Duration: 3.770s
  Energy:   0.0312

...

Summary:
  Total audio duration: 15.234s
  Total speech duration: 12.450s
  Speech ratio: 81.7%
  Silence duration: 2.784s
  Number of segments: 5
  Average segment duration: 2.490s
```

## Creating Your Own Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/sultanfariz/go-vad/pkg/vad"
)

func main() {
    // 1. Load audio
    audioData, err := vad.LoadAudioFile("audio.wav")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create VAD instance
    vadInstance := vad.NewAdaptiveVAD(nil)

    // 3. Detect speech
    segments := vadInstance.DetectSpeech(audioData)

    // 4. Process results
    for _, seg := range segments {
        fmt.Printf("Speech: %.2fs - %.2fs\n",
            seg.Start.Seconds(),
            seg.End.Seconds())
    }
}
```

## Supported Audio Formats

- **WAV**: Native support (no external dependencies)
- **MP3, FLAC, OGG, etc.**: Requires ffmpeg

To install ffmpeg:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg

# Windows
# Download from https://ffmpeg.org/download.html
```

## Tips

1. **For quiet recordings**: Lower the `EnergyThreshold` in config
2. **For noisy environments**: Increase `EnergyThreshold` or use Adaptive VAD
3. **For fragmented results**: Decrease `MinSpeechDuration` or `HopSize`
4. **For better accuracy**: Use Adaptive VAD (recommended for most cases)
5. **For consistent behavior**: Use Basic VAD with well-tuned thresholds

## Troubleshooting

**"failed to load audio"**: Check file path and format
**"ffmpeg not found"**: Install ffmpeg for non-WAV files
**"no segments detected"**: Try lowering `EnergyThreshold` or check audio levels
**"too many segments"**: Increase `EnergyThreshold` or `MinSpeechDuration`
