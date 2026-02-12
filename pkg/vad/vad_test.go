package vad

import (
	"math"
	"testing"
	"time"
)

// TestCalculateEnergy tests the RMS energy calculation
func TestCalculateEnergy(t *testing.T) {
	tests := []struct {
		name     string
		frame    []float64
		expected float64
	}{
		{
			name:     "empty frame",
			frame:    []float64{},
			expected: 0.0,
		},
		{
			name:     "zero frame",
			frame:    []float64{0, 0, 0, 0},
			expected: 0.0,
		},
		{
			name:     "constant frame",
			frame:    []float64{0.5, 0.5, 0.5, 0.5},
			expected: 0.5,
		},
		{
			name:     "varying frame",
			frame:    []float64{0.1, 0.2, 0.3, 0.4},
			expected: math.Sqrt((0.01 + 0.04 + 0.09 + 0.16) / 4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateEnergy(tt.frame)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("calculateEnergy() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestCalculateZCR tests the zero crossing rate calculation
func TestCalculateZCR(t *testing.T) {
	tests := []struct {
		name     string
		frame    []float64
		expected float64
	}{
		{
			name:     "empty frame",
			frame:    []float64{},
			expected: 0.0,
		},
		{
			name:     "single sample",
			frame:    []float64{0.5},
			expected: 0.0,
		},
		{
			name:     "no crossings",
			frame:    []float64{0.1, 0.2, 0.3, 0.4},
			expected: 0.0,
		},
		{
			name:     "one crossing",
			frame:    []float64{0.1, 0.2, -0.1, -0.2},
			expected: 0.25, // 1 crossing / 4 samples
		},
		{
			name:     "alternating",
			frame:    []float64{0.1, -0.1, 0.1, -0.1},
			expected: 0.75, // 3 crossings / 4 samples
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateZCR(tt.frame)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("calculateZCR() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestMedianFilter tests the median filtering function
func TestMedianFilter(t *testing.T) {
	tests := []struct {
		name       string
		data       []bool
		windowSize int
		expected   []bool
	}{
		{
			name:       "empty data",
			data:       []bool{},
			windowSize: 3,
			expected:   []bool{},
		},
		{
			name:       "window size 0",
			data:       []bool{true, false, true},
			windowSize: 0,
			expected:   []bool{true, false, true},
		},
		{
			name:       "remove single false positive",
			data:       []bool{false, false, true, false, false},
			windowSize: 3,
			expected:   []bool{false, false, false, false, false},
		},
		{
			name:       "remove single false negative",
			data:       []bool{true, true, false, true, true},
			windowSize: 3,
			expected:   []bool{true, true, true, true, true},
		},
		{
			name:       "majority wins",
			data:       []bool{true, true, true, false, false},
			windowSize: 3,
			expected:   []bool{true, true, true, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := medianFilter(tt.data, tt.windowSize)
			if len(result) != len(tt.expected) {
				t.Errorf("medianFilter() length = %v, expected %v", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("medianFilter()[%d] = %v, expected %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestAudioDataGetMonoSamples tests mono conversion
func TestAudioDataGetMonoSamples(t *testing.T) {
	tests := []struct {
		name     string
		audio    *AudioData
		expected []float64
	}{
		{
			name: "already mono",
			audio: &AudioData{
				Samples:  []float64{0.1, 0.2, 0.3, 0.4},
				Channels: 1,
			},
			expected: []float64{0.1, 0.2, 0.3, 0.4},
		},
		{
			name: "stereo to mono",
			audio: &AudioData{
				Samples:  []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
				Channels: 2,
			},
			expected: []float64{0.15, 0.35, 0.55}, // (0.1+0.2)/2, (0.3+0.4)/2, (0.5+0.6)/2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.audio.GetMonoSamples()
			if len(result) != len(tt.expected) {
				t.Errorf("GetMonoSamples() length = %v, expected %v", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if math.Abs(result[i]-tt.expected[i]) > 1e-6 {
					t.Errorf("GetMonoSamples()[%d] = %v, expected %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestAudioDataGetFrames tests frame extraction
func TestAudioDataGetFrames(t *testing.T) {
	audio := &AudioData{
		Samples:    []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		Channels:   1,
		SampleRate: 10, // 10 samples per second
	}

	// Frame size: 0.3s = 3 samples
	// Hop size: 0.2s = 2 samples
	frames := audio.GetFrames(300*time.Millisecond, 200*time.Millisecond)

	expectedFrameCount := 4 // (10 - 3) / 2 + 1 = 4
	if len(frames) != expectedFrameCount {
		t.Errorf("GetFrames() returned %d frames, expected %d", len(frames), expectedFrameCount)
	}

	// Check first frame
	expectedFirst := []float64{0.1, 0.2, 0.3}
	for i := range expectedFirst {
		if math.Abs(frames[0][i]-expectedFirst[i]) > 1e-6 {
			t.Errorf("First frame[%d] = %v, expected %v", i, frames[0][i], expectedFirst[i])
		}
	}

	// Check that frames overlap correctly
	// Second frame should start at position 2
	expectedSecond := []float64{0.3, 0.4, 0.5}
	for i := range expectedSecond {
		if math.Abs(frames[1][i]-expectedSecond[i]) > 1e-6 {
			t.Errorf("Second frame[%d] = %v, expected %v", i, frames[1][i], expectedSecond[i])
		}
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.EnergyThreshold != 0.02 {
		t.Errorf("Default EnergyThreshold = %v, expected 0.02", config.EnergyThreshold)
	}
	if config.ZCRThreshold != 0.1 {
		t.Errorf("Default ZCRThreshold = %v, expected 0.1", config.ZCRThreshold)
	}
	if config.MinSpeechDuration != 300*time.Millisecond {
		t.Errorf("Default MinSpeechDuration = %v, expected 300ms", config.MinSpeechDuration)
	}
	if config.FrameSize != 25*time.Millisecond {
		t.Errorf("Default FrameSize = %v, expected 25ms", config.FrameSize)
	}
	if config.HopSize != 10*time.Millisecond {
		t.Errorf("Default HopSize = %v, expected 10ms", config.HopSize)
	}
}

// TestNewVAD tests VAD constructor
func TestNewVAD(t *testing.T) {
	// Test with nil config
	vad1 := NewVAD(nil)
	if vad1 == nil {
		t.Error("NewVAD(nil) returned nil")
	}
	if vad1.config == nil {
		t.Error("NewVAD(nil) has nil config")
	}

	// Test with custom config
	config := &Config{
		EnergyThreshold: 0.05,
		ZCRThreshold:    0.15,
	}
	vad2 := NewVAD(config)
	if vad2 == nil {
		t.Error("NewVAD(config) returned nil")
	}
	if vad2.config.EnergyThreshold != 0.05 {
		t.Errorf("VAD config EnergyThreshold = %v, expected 0.05", vad2.config.EnergyThreshold)
	}
}

// TestNewAdaptiveVAD tests Adaptive VAD constructor
func TestNewAdaptiveVAD(t *testing.T) {
	// Test with nil config
	vad1 := NewAdaptiveVAD(nil)
	if vad1 == nil {
		t.Error("NewAdaptiveVAD(nil) returned nil")
	}
	if vad1.config == nil {
		t.Error("NewAdaptiveVAD(nil) has nil config")
	}

	// Test with custom config
	config := &Config{
		EnergyThreshold: 0.05,
		ZCRThreshold:    0.15,
	}
	vad2 := NewAdaptiveVAD(config)
	if vad2 == nil {
		t.Error("NewAdaptiveVAD(config) returned nil")
	}
	if vad2.config.EnergyThreshold != 0.05 {
		t.Errorf("AdaptiveVAD config EnergyThreshold = %v, expected 0.05", vad2.config.EnergyThreshold)
	}
}

// TestVADDetectSpeech_EmptyAudio tests VAD with empty audio
func TestVADDetectSpeech_EmptyAudio(t *testing.T) {
	vad := NewVAD(nil)

	// Test with nil audio
	segments := vad.DetectSpeech(nil)
	if segments != nil {
		t.Error("DetectSpeech(nil) should return nil")
	}

	// Test with empty samples
	emptyAudio := &AudioData{
		Samples:    []float64{},
		SampleRate: 16000,
		Channels:   1,
	}
	segments = vad.DetectSpeech(emptyAudio)
	if segments != nil {
		t.Error("DetectSpeech(empty) should return nil")
	}
}

// TestAdaptiveVADDetectSpeech_EmptyAudio tests Adaptive VAD with empty audio
func TestAdaptiveVADDetectSpeech_EmptyAudio(t *testing.T) {
	vad := NewAdaptiveVAD(nil)

	// Test with nil audio
	segments := vad.DetectSpeech(nil)
	if segments != nil {
		t.Error("DetectSpeech(nil) should return nil")
	}

	// Test with empty samples
	emptyAudio := &AudioData{
		Samples:    []float64{},
		SampleRate: 16000,
		Channels:   1,
	}
	segments = vad.DetectSpeech(emptyAudio)
	if segments != nil {
		t.Error("DetectSpeech(empty) should return nil")
	}
}
