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
		fmt.Println("Usage: go run main.go <audio_file> [basic|adaptive]")
		fmt.Println("Example: go run main.go conversation.wav adaptive")
		os.Exit(1)
	}

	audioFile := os.Args[1]
	vadType := "basic"
	if len(os.Args) >= 3 {
		vadType = os.Args[2]
	}

	// Load entire audio file for simulation
	// In real streaming, you'd receive chunks from microphone/network
	audioData, err := vad.LoadAudioFile(audioFile)
	if err != nil {
		log.Fatalf("Failed to load audio: %v", err)
	}

	fmt.Printf("Streaming VAD Demo\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Audio File: %s\n", audioFile)
	fmt.Printf("Duration: %v\n", audioData.Duration)
	fmt.Printf("Sample Rate: %d Hz\n", audioData.SampleRate)
	fmt.Printf("VAD Type: %s\n\n", vadType)

	// Create streaming VAD based on type
	config := vad.DefaultConfig()
	var segments []vad.SpeechSegment

	monoSamples := audioData.GetMonoSamples()

	// Simulate streaming by processing in chunks
	chunkDuration := 100 * time.Millisecond
	chunkSize := int(float64(audioData.SampleRate) * chunkDuration.Seconds())

	fmt.Printf("Processing audio in %dms chunks...\n\n", chunkDuration.Milliseconds())

	if vadType == "adaptive" {
		// Use adaptive streaming VAD
		streamVAD := vad.NewStreamingAdaptiveVAD(config, audioData.SampleRate)
		segments = processStreaming(streamVAD, monoSamples, chunkSize)
	} else {
		// Use basic streaming VAD
		streamVAD := vad.NewStreamingVAD(config, audioData.SampleRate)
		segments = processStreaming(streamVAD, monoSamples, chunkSize)
	}

	// Print summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total segments detected: %d\n", len(segments))

	if len(segments) > 0 {
		totalSpeech := 0.0
		for i, seg := range segments {
			totalSpeech += seg.Duration.Seconds()
			fmt.Printf("  Segment %d: %.3fs - %.3fs (%.3fs, energy: %.4f)\n",
				i+1, seg.Start.Seconds(), seg.End.Seconds(),
				seg.Duration.Seconds(), seg.Energy)
		}

		fmt.Printf("\nTotal speech duration: %.3fs\n", totalSpeech)
		fmt.Printf("Speech ratio: %.1f%%\n",
			(totalSpeech/audioData.Duration.Seconds())*100)
	}
}

// Processor interface for both streaming VAD types
type Processor interface {
	ProcessChunk(samples []float64) vad.StreamEvent
	Flush() vad.StreamEvent
}

func processStreaming(processor Processor, samples []float64, chunkSize int) []vad.SpeechSegment {
	segments := []vad.SpeechSegment{}

	// Process chunks
	for offset := 0; offset < len(samples); offset += chunkSize {
		end := offset + chunkSize
		if end > len(samples) {
			end = len(samples)
		}

		chunk := samples[offset:end]
		event := processor.ProcessChunk(chunk)

		switch event.Type {
		case vad.EventSpeechStarted:
			fmt.Printf("[%.3fs] 🎤 Speech started\n", event.Timestamp.Seconds())

		case vad.EventSpeechEnded:
			fmt.Printf("[%.3fs] ⏸️  Speech ended - Duration: %.3fs, Energy: %.4f\n",
				event.Timestamp.Seconds(),
				event.Segment.Duration.Seconds(),
				event.Segment.Energy)
			segments = append(segments, *event.Segment)
		}
	}

	// Flush remaining samples
	finalEvent := processor.Flush()
	if finalEvent.Type == vad.EventSpeechEnded && finalEvent.Segment != nil {
		fmt.Printf("[%.3fs] ⏸️  Speech ended (flush) - Duration: %.3fs, Energy: %.4f\n",
			finalEvent.Timestamp.Seconds(),
			finalEvent.Segment.Duration.Seconds(),
			finalEvent.Segment.Energy)
		segments = append(segments, *finalEvent.Segment)
	}

	return segments
}
