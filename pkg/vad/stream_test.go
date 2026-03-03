package vad

import (
	"math"
	"testing"
	"time"
)

// Test helper: Generate synthetic speech signal (high energy, low ZCR)
func generateSyntheticSpeech(length int, sampleRate int) []float64 {
	samples := make([]float64, length)
	for i := range samples {
		// Sine wave at 100Hz (low frequency = low ZCR)
		samples[i] = 0.1 * math.Sin(2*math.Pi*100*float64(i)/float64(sampleRate))
	}
	return samples
}

// Test helper: Generate silence (low energy)
func generateSilence(length int) []float64 {
	return make([]float64, length)
}

func TestNewStreamingVAD(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		sampleRate int
		wantNil    bool
	}{
		{
			name:       "with nil config",
			config:     nil,
			sampleRate: 16000,
			wantNil:    false,
		},
		{
			name: "with custom config",
			config: &Config{
				EnergyThreshold:   0.03,
				ZCRThreshold:      0.15,
				MinSpeechDuration: 200 * time.Millisecond,
				FrameSize:         25 * time.Millisecond,
				HopSize:           10 * time.Millisecond,
			},
			sampleRate: 16000,
			wantNil:    false,
		},
		{
			name:       "with 8kHz sample rate",
			config:     nil,
			sampleRate: 8000,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vad := NewStreamingVAD(tt.config, tt.sampleRate)
			if (vad == nil) != tt.wantNil {
				t.Errorf("NewStreamingVAD() = %v, wantNil %v", vad, tt.wantNil)
			}
			if vad != nil && vad.sampleRate != tt.sampleRate {
				t.Errorf("sampleRate = %v, want %v", vad.sampleRate, tt.sampleRate)
			}
		})
	}
}

func TestNewStreamingAdaptiveVAD(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		sampleRate int
		wantNil    bool
	}{
		{
			name:       "with nil config",
			config:     nil,
			sampleRate: 16000,
			wantNil:    false,
		},
		{
			name: "with custom config",
			config: &Config{
				EnergyThreshold:   0.03,
				ZCRThreshold:      0.15,
				MinSpeechDuration: 200 * time.Millisecond,
				FrameSize:         25 * time.Millisecond,
				HopSize:           10 * time.Millisecond,
			},
			sampleRate: 16000,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vad := NewStreamingAdaptiveVAD(tt.config, tt.sampleRate)
			if (vad == nil) != tt.wantNil {
				t.Errorf("NewStreamingAdaptiveVAD() = %v, wantNil %v", vad, tt.wantNil)
			}
			if vad != nil && vad.sampleRate != tt.sampleRate {
				t.Errorf("sampleRate = %v, want %v", vad.sampleRate, tt.sampleRate)
			}
			if vad != nil && len(vad.energyHistory) != 0 {
				t.Errorf("energyHistory should be empty initially")
			}
		})
	}
}

func TestStreamingVAD_ProcessChunk_EmptyChunk(t *testing.T) {
	vad := NewStreamingVAD(nil, 16000)
	event := vad.ProcessChunk([]float64{})

	if event.Type != EventNone {
		t.Errorf("ProcessChunk(empty) should return EventNone, got %v", event.Type)
	}
}

func TestStreamingVAD_ProcessChunk_SpeechDetection(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 100 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Create synthetic speech: 500ms of high energy, low ZCR
	chunkDuration := 100 * time.Millisecond
	chunkSize := int(float64(sampleRate) * chunkDuration.Seconds())

	speechStarted := false
	speechEnded := false

	// Process 5 chunks of speech (500ms total)
	for i := 0; i < 5; i++ {
		chunk := generateSyntheticSpeech(chunkSize, sampleRate)
		event := vad.ProcessChunk(chunk)

		if event.Type == EventSpeechStarted {
			speechStarted = true
		}
		if event.Type == EventSpeechEnded {
			speechEnded = true
		}
	}

	// Add silence to trigger speech end
	silenceChunk := generateSilence(chunkSize)
	for i := 0; i < 3; i++ {
		event := vad.ProcessChunk(silenceChunk)
		if event.Type == EventSpeechEnded {
			speechEnded = true
		}
	}

	if !speechStarted {
		t.Error("Expected EventSpeechStarted")
	}
	if !speechEnded {
		t.Error("Expected EventSpeechEnded")
	}
}

func TestStreamingVAD_VariableChunkSizes(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Test different chunk sizes
	chunkSizes := []int{
		int(float64(sampleRate) * 0.01), // 10ms
		int(float64(sampleRate) * 0.05), // 50ms
		int(float64(sampleRate) * 0.1),  // 100ms
		int(float64(sampleRate) * 0.5),  // 500ms
	}

	for _, size := range chunkSizes {
		vad.Reset()
		chunk := generateSyntheticSpeech(size, sampleRate)
		event := vad.ProcessChunk(chunk)

		// Should not panic or error
		if event.Timestamp < 0 {
			t.Errorf("Invalid timestamp for chunk size %d", size)
		}
	}
}

func TestStreamingVAD_BufferBoundaries(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Process chunks smaller than frame size
	smallChunk := generateSyntheticSpeech(100, sampleRate)

	// Should accumulate in buffer
	event := vad.ProcessChunk(smallChunk)
	if event.Type != EventNone {
		t.Error("Small chunk should not trigger event immediately")
	}

	if len(vad.sampleBuffer) != 100 {
		t.Errorf("Buffer should contain 100 samples, got %d", len(vad.sampleBuffer))
	}
}

func TestStreamingVAD_Flush(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 50 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Process speech
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*0.2), sampleRate)
	vad.ProcessChunk(chunk)

	// Flush should finalize any remaining segment
	_ = vad.Flush()

	if vad.inSpeech {
		t.Error("After Flush(), inSpeech should be false")
	}

	if len(vad.sampleBuffer) != 0 {
		t.Errorf("After Flush(), sampleBuffer should be empty, got %d", len(vad.sampleBuffer))
	}
}

func TestStreamingVAD_Reset(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Process some audio
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*0.1), sampleRate)
	vad.ProcessChunk(chunk)

	// Reset
	vad.Reset()

	// Check all state is cleared
	if vad.totalSamples != 0 {
		t.Error("totalSamples should be 0 after Reset()")
	}
	if len(vad.sampleBuffer) != 0 {
		t.Error("sampleBuffer should be empty after Reset()")
	}
	if vad.inSpeech {
		t.Error("inSpeech should be false after Reset()")
	}
	if len(vad.recentDecisions) != 0 {
		t.Error("recentDecisions should be empty after Reset()")
	}
}

func TestStreamingVAD_MinimumDuration(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 300 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	// Process very short speech (100ms - below minimum)
	shortSpeech := generateSyntheticSpeech(int(float64(sampleRate)*0.1), sampleRate)
	vad.ProcessChunk(shortSpeech)

	// Add silence
	silence := generateSilence(int(float64(sampleRate) * 0.1))
	for i := 0; i < 5; i++ {
		event := vad.ProcessChunk(silence)

		// Should not emit segment because duration < MinSpeechDuration
		if event.Type == EventSpeechEnded && event.Segment != nil {
			if event.Segment.Duration < config.MinSpeechDuration {
				t.Error("Should not emit segment shorter than MinSpeechDuration")
			}
		}
	}
}

func TestStreamingVAD_StateTransitions(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 50 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	transitions := []struct {
		signal    func(int) []float64
		chunkSize int
		expected  StreamEventType
	}{
		{
			signal: func(size int) []float64 {
				return generateSyntheticSpeech(size, sampleRate)
			},
			chunkSize: int(float64(sampleRate) * 0.2),
			expected:  EventSpeechStarted,
		},
		{
			signal:    generateSilence,
			chunkSize: int(float64(sampleRate) * 0.2),
			expected:  EventSpeechEnded,
		},
	}

	for i, tt := range transitions {
		chunk := tt.signal(tt.chunkSize)

		eventFound := false
		for j := 0; j < 5; j++ {
			event := vad.ProcessChunk(chunk)
			if event.Type == tt.expected {
				eventFound = true
				break
			}
		}

		if !eventFound && tt.expected != EventNone {
			t.Errorf("Transition %d: expected %v event", i, tt.expected)
		}
	}
}

func TestStreamingAdaptiveVAD_ProcessChunk(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 100 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingAdaptiveVAD(config, sampleRate)

	// Process some audio to build history
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*0.5), sampleRate)
	event := vad.ProcessChunk(chunk)

	// Should not panic
	if event.Timestamp < 0 {
		t.Error("Invalid timestamp")
	}
}

func TestStreamingAdaptiveVAD_FeatureHistory(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingAdaptiveVAD(config, sampleRate)

	// Process enough audio to fill history
	// 100 frames * 10ms hop = 1 second
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*1.5), sampleRate)
	vad.ProcessChunk(chunk)

	// History should be capped at 100
	if len(vad.energyHistory) > 100 {
		t.Errorf("Energy history should be capped at 100, got %d", len(vad.energyHistory))
	}
	if len(vad.zcrHistory) > 100 {
		t.Errorf("ZCR history should be capped at 100, got %d", len(vad.zcrHistory))
	}
}

func TestStreamingAdaptiveVAD_AdaptiveThresholds(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingAdaptiveVAD(config, sampleRate)

	// Initially should use config thresholds
	threshold := vad.calculateDynamicThresholdStreaming()
	if threshold != config.EnergyThreshold {
		t.Errorf("With no history, should use config threshold")
	}

	// Build history with speech
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*1.0), sampleRate)
	vad.ProcessChunk(chunk)

	// Now should calculate from history
	if len(vad.energyHistory) >= 20 {
		threshold = vad.calculateDynamicThresholdStreaming()
		// Threshold should be calculated from history, may differ from config
		// Just check it's within reasonable bounds
		if threshold < 0.0001 || threshold > 0.08 {
			t.Errorf("Threshold %f outside reasonable bounds", threshold)
		}
	}
}

func TestStreamingAdaptiveVAD_Reset(t *testing.T) {
	config := DefaultConfig()
	sampleRate := 16000
	vad := NewStreamingAdaptiveVAD(config, sampleRate)

	// Build up history
	chunk := generateSyntheticSpeech(int(float64(sampleRate)*1.0), sampleRate)
	vad.ProcessChunk(chunk)

	// Reset
	vad.Reset()

	// Check all state is cleared
	if len(vad.energyHistory) != 0 {
		t.Error("energyHistory should be empty after Reset()")
	}
	if len(vad.zcrHistory) != 0 {
		t.Error("zcrHistory should be empty after Reset()")
	}
	if vad.totalSamples != 0 {
		t.Error("totalSamples should be 0 after Reset()")
	}
}

func TestStreamingVAD_MultipleSegments(t *testing.T) {
	config := &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 100 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}

	sampleRate := 16000
	vad := NewStreamingVAD(config, sampleRate)

	segmentCount := 0

	// Alternate between speech and silence multiple times
	for i := 0; i < 3; i++ {
		// Speech
		speech := generateSyntheticSpeech(int(float64(sampleRate)*0.2), sampleRate)
		event := vad.ProcessChunk(speech)
		if event.Type == EventSpeechStarted {
			segmentCount++
		}

		// Silence
		silence := generateSilence(int(float64(sampleRate) * 0.2))
		for j := 0; j < 3; j++ {
			vad.ProcessChunk(silence)
		}
	}

	if segmentCount < 2 {
		t.Errorf("Expected at least 2 speech segments, got %d", segmentCount)
	}
}

func TestStreamEventType_String(t *testing.T) {
	tests := []struct {
		eventType StreamEventType
		want      string
	}{
		{EventNone, "None"},
		{EventSpeechStarted, "SpeechStarted"},
		{EventSpeechEnded, "SpeechEnded"},
		{StreamEventType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
