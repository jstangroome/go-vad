package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sultanfariz/go-vad/pkg/vad"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <audio_file>")
		fmt.Println("Example: go run main.go conversation.wav")
		os.Exit(1)
	}

	audioFile := os.Args[1]

	// Load audio file
	fmt.Printf("Loading audio file: %s\n", audioFile)
	audioData, err := vad.LoadAudioFile(audioFile)
	if err != nil {
		log.Fatalf("Failed to load audio: %v", err)
	}

	fmt.Printf("Audio loaded successfully:\n")
	fmt.Printf("  File: %s\n", audioData.FileName)
	fmt.Printf("  Duration: %v\n", audioData.Duration)
	fmt.Printf("  Sample Rate: %d Hz\n", audioData.SampleRate)
	fmt.Printf("  Channels: %d\n", audioData.Channels)
	fmt.Println()

	// Create custom configuration for adaptive VAD
	config := &vad.Config{
		EnergyThreshold:   0.02,  // Used as fallback
		ZCRThreshold:      0.1,   // Used as fallback
		MinSpeechDuration: 300 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	// Create adaptive VAD
	vadInstance := vad.NewAdaptiveVAD(config)

	// Detect speech segments
	fmt.Println("Running Adaptive VAD...")
	fmt.Println("(Adaptive VAD automatically adjusts thresholds based on audio characteristics)")
	fmt.Println()

	segments := vadInstance.DetectSpeech(audioData)

	// Print results
	fmt.Printf("Detected %d speech segments:\n", len(segments))
	fmt.Println("---")

	totalSpeechDuration := 0.0
	for i, seg := range segments {
		fmt.Printf("Segment %d:\n", i+1)
		fmt.Printf("  Start:    %.3fs\n", seg.Start.Seconds())
		fmt.Printf("  End:      %.3fs\n", seg.End.Seconds())
		fmt.Printf("  Duration: %.3fs\n", seg.Duration.Seconds())
		fmt.Printf("  Energy:   %.4f\n", seg.Energy)
		fmt.Println()

		totalSpeechDuration += seg.Duration.Seconds()
	}

	// Print summary statistics
	fmt.Println("Summary:")
	fmt.Printf("  Total audio duration: %.3fs\n", audioData.Duration.Seconds())
	fmt.Printf("  Total speech duration: %.3fs\n", totalSpeechDuration)
	fmt.Printf("  Speech ratio: %.1f%%\n", (totalSpeechDuration/audioData.Duration.Seconds())*100)
	fmt.Printf("  Silence duration: %.3fs\n", audioData.Duration.Seconds()-totalSpeechDuration)
	fmt.Printf("  Number of segments: %d\n", len(segments))

	if len(segments) > 0 {
		avgDuration := totalSpeechDuration / float64(len(segments))
		fmt.Printf("  Average segment duration: %.3fs\n", avgDuration)
	}
}
