package vad

import (
	"testing"
	"time"
)

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
