package main

import (
	"fmt"
	"log"
	"os"

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

	fmt.Printf("Audio loaded: %s (%.2fs, %d Hz, %d channels)\n\n",
		audioData.FileName,
		audioData.Duration.Seconds(),
		audioData.SampleRate,
		audioData.Channels)

	// Run Basic VAD
	fmt.Println("=== BASIC VAD (Fixed Thresholds) ===")
	basicVAD := vad.NewVAD(nil)
	basicSegments := basicVAD.DetectSpeech(audioData)

	basicSpeechDuration := 0.0
	for _, seg := range basicSegments {
		basicSpeechDuration += seg.Duration.Seconds()
	}

	fmt.Printf("Segments detected: %d\n", len(basicSegments))
	fmt.Printf("Total speech: %.3fs (%.1f%%)\n",
		basicSpeechDuration,
		(basicSpeechDuration/audioData.Duration.Seconds())*100)

	if len(basicSegments) > 0 {
		fmt.Printf("Average segment: %.3fs\n", basicSpeechDuration/float64(len(basicSegments)))
	}
	fmt.Println()

	// Run Adaptive VAD
	fmt.Println("=== ADAPTIVE VAD (Dynamic Thresholds) ===")
	adaptiveVAD := vad.NewAdaptiveVAD(nil)
	adaptiveSegments := adaptiveVAD.DetectSpeech(audioData)

	adaptiveSpeechDuration := 0.0
	for _, seg := range adaptiveSegments {
		adaptiveSpeechDuration += seg.Duration.Seconds()
	}

	fmt.Printf("Segments detected: %d\n", len(adaptiveSegments))
	fmt.Printf("Total speech: %.3fs (%.1f%%)\n",
		adaptiveSpeechDuration,
		(adaptiveSpeechDuration/audioData.Duration.Seconds())*100)

	if len(adaptiveSegments) > 0 {
		fmt.Printf("Average segment: %.3fs\n", adaptiveSpeechDuration/float64(len(adaptiveSegments)))
	}
	fmt.Println()

	// Comparison
	fmt.Println("=== COMPARISON ===")
	segmentDiff := len(adaptiveSegments) - len(basicSegments)
	durationDiff := adaptiveSpeechDuration - basicSpeechDuration

	fmt.Printf("Segment count difference: %+d\n", segmentDiff)
	fmt.Printf("Speech duration difference: %+.3fs\n", durationDiff)

	if segmentDiff > 0 {
		fmt.Println("\nAdaptive VAD detected more segments (possibly more sensitive to quieter speech)")
	} else if segmentDiff < 0 {
		fmt.Println("\nBasic VAD detected more segments (possibly more false positives)")
	} else {
		fmt.Println("\nBoth methods detected the same number of segments")
	}

	fmt.Println("\nNote: Adaptive VAD typically performs better in varying audio conditions")
	fmt.Println("      as it automatically adjusts to local audio characteristics.")
}
