# go-vad

A pure Go implementation of Voice Activity Detection (VAD) using energy-based and zero-crossing rate features.

## Features

- **No external ML dependencies** - Pure signal processing approach
- **Dual VAD implementations**:
  - **Basic VAD**: Fixed thresholds for consistent behavior
  - **Adaptive VAD**: Dynamic thresholds that adapt to audio characteristics (recommended)
- **Dual-feature approach**: Combines RMS energy and Zero Crossing Rate (ZCR)
- **Frame-based processing**: Analyzes audio in overlapping windows
- **Post-processing**: Median filtering and duration-based filtering
- **Configurable parameters**: All thresholds and timing parameters adjustable
- **Sample rate agnostic**: Works with any sample rate (8kHz-48kHz tested)
- **Multiple format support**: WAV (native), MP3, FLAC, etc. (via ffmpeg)

## Installation

```bash
go get github.com/sultanfariz/go-vad/pkg/vad
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/sultanfariz/go-vad/pkg/vad"
)

func main() {
    // Load audio file
    audioData, err := vad.LoadAudioFile("conversation.wav")
    if err != nil {
        log.Fatal(err)
    }

    // Create adaptive VAD with default config
    vadInstance := vad.NewAdaptiveVAD(nil)

    // Detect speech segments
    segments := vadInstance.DetectSpeech(audioData)

    // Print results
    fmt.Printf("Detected %d speech segments:\n", len(segments))
    for i, seg := range segments {
        fmt.Printf("Segment %d: %.2fs - %.2fs (duration: %.2fs, energy: %.4f)\n",
            i+1, seg.Start.Seconds(), seg.End.Seconds(),
            seg.Duration.Seconds(), seg.Energy)
    }
}
```

## Usage

### Basic VAD

Uses fixed thresholds defined in configuration:

```go
// Create configuration
config := &vad.Config{
    EnergyThreshold:   0.02,
    ZCRThreshold:      0.1,
    MinSpeechDuration: 300 * time.Millisecond,
    FrameSize:         25 * time.Millisecond,
    HopSize:           10 * time.Millisecond,
}

// Create basic VAD
vadInstance := vad.NewVAD(config)

// Detect speech
segments := vadInstance.DetectSpeech(audioData)
```

### Adaptive VAD (Recommended)

Automatically adjusts thresholds based on local audio characteristics:

```go
// Use default config (recommended for most cases)
vadInstance := vad.NewAdaptiveVAD(nil)

// Or customize
config := vad.DefaultConfig()
config.MinSpeechDuration = 500 * time.Millisecond
vadInstance = vad.NewAdaptiveVAD(config)

// Detect speech
segments := vadInstance.DetectSpeech(audioData)
```

### Configuration Parameters

```go
type Config struct {
    // Energy threshold for speech detection
    // Range: 0.0 - 1.0 (normalized audio)
    // Default: 0.02
    // Lower = more sensitive, Higher = less sensitive
    EnergyThreshold float64

    // Zero Crossing Rate threshold
    // Range: 0.0 - 1.0
    // Default: 0.1
    // Speech typically < 0.1, noise > 0.1
    ZCRThreshold float64

    // Minimum speech duration to keep
    // Default: 300ms
    // Filters out short noise bursts
    MinSpeechDuration time.Duration

    // Frame size for analysis window
    // Default: 25ms
    // Typical range: 10-30ms
    FrameSize time.Duration

    // Hop size (stride) between frames
    // Default: 10ms
    // Creates 60% overlap with 25ms frames
    HopSize time.Duration
}
```

### Custom Configuration

```go
// For quiet speech (low volume)
config := &vad.Config{
    EnergyThreshold:   0.008,
    ZCRThreshold:      0.15,
    MinSpeechDuration: 250 * time.Millisecond,
    FrameSize:         25 * time.Millisecond,
    HopSize:           10 * time.Millisecond,
}

// For noisy environments
config := &vad.Config{
    EnergyThreshold:   0.04,
    ZCRThreshold:      0.08,
    MinSpeechDuration: 400 * time.Millisecond,
    FrameSize:         30 * time.Millisecond,
    HopSize:           15 * time.Millisecond,
}

// For high time resolution (fast speech)
config := &vad.Config{
    EnergyThreshold:   0.02,
    ZCRThreshold:      0.1,
    MinSpeechDuration: 200 * time.Millisecond,
    FrameSize:         20 * time.Millisecond,
    HopSize:           5 * time.Millisecond,
}
```

## How It Works

### Algorithm Overview

1. **Frame Extraction**: Audio is divided into overlapping frames (default: 25ms windows with 10ms hop)
2. **Feature Calculation**:
   - **Energy (RMS)**: `E = sqrt(sum(x²) / N)` - Measures signal amplitude
   - **Zero Crossing Rate**: `ZCR = (sign changes) / N` - Measures signal frequency
3. **Thresholding**:
   - Basic VAD: Fixed thresholds from config
   - Adaptive VAD: Dynamic thresholds from local percentiles
4. **Decision Rule**: Speech detected if `(energy > threshold) AND (zcr < threshold)`
5. **Post-processing**:
   - Median filter: Smooths isolated false positives/negatives
   - Segment merging: Groups consecutive speech frames
   - Duration filter: Removes segments shorter than minimum

### Why Energy + ZCR?

- **Speech**: High energy, low ZCR (periodic, harmonic structure)
- **Silence**: Low energy, low ZCR
- **Noise**: Variable energy, high ZCR (random, non-periodic)

Combining both features reduces false positives significantly.

### Adaptive VAD Advantages

- **Adapts to volume changes**: Uses percentile-based thresholds
- **Handles different recording conditions**: No manual tuning needed
- **Robust to background noise**: Local statistics adapt to noise floor
- **Bounded thresholds**: Prevents extreme values with safety limits

## Audio Format Support

### Native Support

- **WAV**: 16-bit PCM (no external dependencies)

### Via ffmpeg

Requires ffmpeg installed on system:

- MP3, FLAC, OGG, OPUS, M4A, AAC, WEBM

The library automatically converts non-WAV formats to WAV using ffmpeg.

## Data Structures

### AudioData

```go
type AudioData struct {
    Samples    []float64     // Normalized to [-1.0, 1.0]
    SampleRate int           // Hz (e.g., 16000, 44100)
    Channels   int           // 1=mono, 2=stereo
    Duration   time.Duration // Total audio duration
    FileName   string        // Original filename
}
```

### SpeechSegment

```go
type SpeechSegment struct {
    Start    time.Duration // Segment start time
    End      time.Duration // Segment end time
    Speaker  string        // Speaker label (optional)
    Energy   float64       // Average RMS energy
    Duration time.Duration // Segment duration (End - Start)
}
```

## Performance

### Computational Complexity

- **Per Frame**: O(N) where N = frame length
- **Total**: O(M × N) where M = number of frames
- **Typical**: 1 minute audio @ 16kHz processes in < 100ms on modern CPU

### Memory Usage

- **AudioData**: ~8 bytes per sample
- **Processing**: Additional ~2x for frame buffers
- **Total**: ~20-30 MB for 1 minute of audio

## Troubleshooting

### Too Many False Positives (Noise as Speech)

**Solutions:**
1. Increase `EnergyThreshold` (e.g., 0.02 → 0.04)
2. Decrease `ZCRThreshold` (e.g., 0.1 → 0.08)
3. Increase `MinSpeechDuration` (e.g., 300ms → 500ms)
4. Use Adaptive VAD instead of Basic VAD

### Too Many False Negatives (Missing Speech)

**Solutions:**
1. Decrease `EnergyThreshold` (e.g., 0.02 → 0.01)
2. Increase `ZCRThreshold` (e.g., 0.1 → 0.15)
3. Decrease `MinSpeechDuration` (e.g., 300ms → 200ms)
4. Check audio normalization

### Segments Too Fragmented

**Solutions:**
1. Decrease `HopSize` for more overlap (e.g., 10ms → 5ms)
2. Increase median filter window size
3. Implement custom segment merging with gap tolerance

## Examples

See the [examples](examples/) directory for complete working examples:

- [basic](examples/basic/main.go): Simple VAD usage
- [adaptive](examples/adaptive/main.go): Adaptive VAD with custom config
- [comparison](examples/comparison/main.go): Compare Basic vs Adaptive VAD

## Use Cases

- Speaker diarization (who spoke when)
- Conversation turn detection
- Audio quality analysis
- Speech/silence segmentation
- Latency measurement in conversational AI
- Preprocessing for speech recognition
- Audio compression (skip silence)

## Technical Details

### Mathematical Formulas

**Energy (RMS)**:
```
E = sqrt((1/N) × Σ(x[i]²))
```

**Zero Crossing Rate**:
```
ZCR = (1/N) × Σ(|sgn(x[i]) - sgn(x[i-1])|)
```

**Frame Timing**:
```
Frame start = frame_index × hop_size
Overlap ratio = (frame_size - hop_size) / frame_size
Default overlap = (25ms - 10ms) / 25ms = 60%
```

**Adaptive Thresholds**:
```
Energy threshold = 25th percentile of local window (bounded: [0.0001, 0.08])
ZCR threshold = 60th percentile of local window (bounded: [0.01, 0.5])
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## References

- Frame-based analysis: Standard approach in speech processing
- Energy-based VAD: Classic algorithm from 1970s speech research
- Zero Crossing Rate: Time-domain feature for speech/noise discrimination
- Median filtering: Common post-processing for binary decisions
