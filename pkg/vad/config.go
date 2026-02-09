package vad

import "time"

// Config holds the configuration parameters for VAD.
type Config struct {
	// EnergyThreshold for speech detection
	// Range: 0.0 - 1.0 (normalized audio range)
	// Default: 0.02
	// Lower = more sensitive (detects quieter speech)
	// Higher = less sensitive (only loud speech)
	EnergyThreshold float64

	// ZCRThreshold is the zero crossing rate threshold
	// Range: 0.0 - 1.0 (fraction of frame length)
	// Default: 0.1
	// Speech typically has ZCR < 0.1
	// Noise has higher ZCR
	ZCRThreshold float64

	// MinSpeechDuration is the minimum speech duration to keep
	// Default: 300ms
	// Filters out short noise bursts
	MinSpeechDuration time.Duration

	// FrameSize is the frame size for analysis
	// Default: 25ms
	// Typical range: 10-30ms
	// Smaller = more time resolution, more computation
	FrameSize time.Duration

	// HopSize is the hop size (stride) between frames
	// Default: 10ms
	// Creates 60% overlap with 25ms frames
	// Smaller = more overlap, smoother results
	HopSize time.Duration
}

// DefaultConfig returns the default VAD configuration.
func DefaultConfig() *Config {
	return &Config{
		EnergyThreshold:   0.02,
		ZCRThreshold:      0.1,
		MinSpeechDuration: 300 * time.Millisecond,
		FrameSize:         25 * time.Millisecond,
		HopSize:           10 * time.Millisecond,
	}
}

// NewConfig creates a new Config with custom values, falling back to defaults.
func NewConfig(energyThreshold, zcrThreshold float64, minSpeechDuration, frameSize, hopSize time.Duration) *Config {
	cfg := DefaultConfig()

	if energyThreshold > 0 {
		cfg.EnergyThreshold = energyThreshold
	}
	if zcrThreshold > 0 {
		cfg.ZCRThreshold = zcrThreshold
	}
	if minSpeechDuration > 0 {
		cfg.MinSpeechDuration = minSpeechDuration
	}
	if frameSize > 0 {
		cfg.FrameSize = frameSize
	}
	if hopSize > 0 {
		cfg.HopSize = hopSize
	}

	return cfg
}
