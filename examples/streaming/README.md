# Streaming VAD Example

This example demonstrates how to use the go-vad streaming API for real-time voice activity detection.

## Overview

The streaming API processes audio incrementally in chunks, making it suitable for:
- Live microphone input
- Network streaming (WebRTC, VoIP)
- Real-time transcription
- Low-latency speech detection

## Running the Example

```bash
# Basic streaming VAD
go run main.go <audio_file>

# Adaptive streaming VAD
go run main.go <audio_file> adaptive
```

### Examples

```bash
# Process with basic streaming VAD
go run main.go ../../testdata/speech.wav

# Process with adaptive streaming VAD (recommended for varying conditions)
go run main.go ../../testdata/speech.wav adaptive
```

## How It Works

The example simulates streaming by:
1. Loading a complete audio file
2. Breaking it into 100ms chunks
3. Processing each chunk sequentially
4. Printing events in real-time as they occur

In a real application, chunks would come from:
- Microphone input buffers
- Network packets
- Audio streaming APIs

## Output

The program prints:
- **Speech Started** events when speech begins
- **Speech Ended** events when speech ends, including:
  - Timestamp
  - Duration
  - Average energy
- Summary statistics at the end

Example output:
```
Streaming VAD Demo
==================

Audio File: conversation.wav
Duration: 10s
Sample Rate: 16000 Hz
VAD Type: basic

Processing audio in 100ms chunks...

[0.350s] 🎤 Speech started
[2.120s] ⏸️  Speech ended - Duration: 1.770s, Energy: 0.0523
[3.890s] 🎤 Speech started
[5.430s] ⏸️  Speech ended - Duration: 1.540s, Energy: 0.0487

=== Summary ===
Total segments detected: 2
  Segment 1: 0.350s - 2.120s (1.770s, energy: 0.0523)
  Segment 2: 3.890s - 5.430s (1.540s, energy: 0.0487)

Total speech duration: 3.310s
Speech ratio: 33.1%
```

## Code Structure

### Key Components

1. **Processor Interface**: Allows using either `StreamingVAD` or `StreamingAdaptiveVAD`
2. **Chunk Processing Loop**: Simulates real-time streaming
3. **Event Handling**: Responds to `SpeechStarted` and `SpeechEnded` events
4. **Flush**: Ensures all buffered audio is processed at the end

### Real-Time Streaming Pattern

```go
// Create streaming VAD
streamVAD := vad.NewStreamingVAD(config, sampleRate)

// Process chunks as they arrive
for chunk := range audioStream {
    event := streamVAD.ProcessChunk(chunk)

    switch event.Type {
    case vad.EventSpeechStarted:
        // Start recording, wake up ASR, etc.

    case vad.EventSpeechEnded:
        // Stop recording, process segment
        segment := event.Segment
        processSegment(segment)
    }
}

// When stream ends
finalEvent := streamVAD.Flush()
```

## Configuration

The example uses default configuration:
- **Energy Threshold**: 0.02 (basic) or adaptive
- **ZCR Threshold**: 0.1 (basic) or adaptive
- **Min Speech Duration**: 300ms
- **Frame Size**: 25ms
- **Hop Size**: 10ms

To use custom settings:
```go
config := &vad.Config{
    EnergyThreshold:   0.03,
    ZCRThreshold:      0.15,
    MinSpeechDuration: 200 * time.Millisecond,
    FrameSize:         25 * time.Millisecond,
    HopSize:           10 * time.Millisecond,
}

streamVAD := vad.NewStreamingVAD(config, sampleRate)
```

## Choosing VAD Type

### Basic Streaming VAD
- Uses fixed thresholds from config
- Lower latency (~50-75ms)
- Best for controlled environments
- Lower memory usage

### Adaptive Streaming VAD
- Dynamically adjusts thresholds
- Handles varying audio levels
- More robust to noise variations
- Requires ~20 frames to "warm up"
- Slightly higher memory (~5KB vs ~3KB)

**Recommendation**: Start with adaptive VAD for most applications.

## Latency Characteristics

- **Minimum Latency**: 1 frame (25ms default)
- **Typical Detection Latency**: 50-75ms (includes median filter smoothing)
- **Chunk Size**: Flexible, typically 10-100ms

Smaller chunks reduce latency but increase processing overhead.

## Memory Usage

Per StreamingVAD instance:
- Basic: ~3KB (sample buffer + state)
- Adaptive: ~5KB (includes feature history)

Both are lightweight enough for real-time applications.

## Error Handling

The streaming API is designed to be robust:
- Handles variable chunk sizes
- Handles empty chunks gracefully
- No assumptions about chunk boundaries
- Automatically buffers partial frames

## See Also

- [Basic VAD Example](../basic/)
- [Adaptive VAD Example](../adaptive/)
- [Comparison Example](../comparison/)
- [Main README](../../README.md)
