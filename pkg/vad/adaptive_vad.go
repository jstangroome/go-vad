package vad

import "sort"

// AdaptiveVAD implements Voice Activity Detection with adaptive thresholds.
// Dynamically adjusts thresholds based on local audio characteristics.
type AdaptiveVAD struct {
	config *Config
}

// NewAdaptiveVAD creates a new adaptive VAD instance with the given configuration.
// If config is nil, uses default configuration.
func NewAdaptiveVAD(config *Config) *AdaptiveVAD {
	if config == nil {
		config = DefaultConfig()
	}
	return &AdaptiveVAD{config: config}
}

// DetectSpeech detects speech segments using adaptive thresholds.
//
// Algorithm:
// 1. Calculate energy and ZCR for all frames
// 2. For each frame, calculate dynamic thresholds from local window
// 3. Apply thresholds: speech if (energy > dynamic_threshold) AND (zcr < dynamic_threshold)
// 4. Apply median filter to smooth results
// 5. Merge consecutive speech frames into segments
// 6. Filter by minimum duration
//
// Advantages over basic VAD:
// - Adapts to varying audio volumes
// - Handles different recording conditions
// - More robust to background noise variations
func (v *AdaptiveVAD) DetectSpeech(audioData *AudioData) []SpeechSegment {
	if audioData == nil || len(audioData.Samples) == 0 {
		return nil
	}

	// Extract frames
	frames := audioData.GetFrames(v.config.FrameSize, v.config.HopSize)
	if len(frames) == 0 {
		return nil
	}

	// Calculate features for all frames
	energies := make([]float64, len(frames))
	zcrs := make([]float64, len(frames))
	isSpeech := make([]bool, len(frames))

	for i, frame := range frames {
		energies[i] = calculateEnergy(frame)
		if v.config.DisableZCR {
			zcrs[i] = -1
		} else {
			zcrs[i] = calculateZCR(frame)
		}
	}

	// Apply adaptive thresholding
	windowSize := 100 // Default window size for local statistics
	for i := range frames {
		// Get local window for dynamic threshold calculation
		energyThreshold := v.calculateDynamicThreshold(energies, i, windowSize)
		var zcrThreshold float64 = 0
		if !v.config.DisableZCR {
			zcrThreshold = v.calculateDynamicZCRThreshold(zcrs, i, windowSize)
		}

		// Dual-threshold decision
		isSpeech[i] = energies[i] > energyThreshold && zcrs[i] < zcrThreshold
	}

	// Apply median filter to smooth detection (window size 5)
	isSpeech = medianFilter(isSpeech, 5)

	// Merge consecutive speech frames into segments
	segments := mergeSegments(isSpeech, audioData, energies, v.config.MinSpeechDuration)

	return segments
}

// calculateDynamicThreshold computes adaptive energy threshold from local window.
// Uses 25th percentile (top 75% considered potential speech).
// Applies bounds to prevent unreasonable thresholds.
func (v *AdaptiveVAD) calculateDynamicThreshold(energies []float64, index, windowSize int) float64 {
	// Calculate window bounds
	halfWindow := windowSize / 2
	start := index - halfWindow
	end := index + halfWindow + 1

	if start < 0 {
		start = 0
	}
	if end > len(energies) {
		end = len(energies)
	}

	// Extract local window
	localEnergies := make([]float64, end-start)
	copy(localEnergies, energies[start:end])

	// Sort to calculate percentile
	sort.Float64s(localEnergies)

	// Get 25th percentile
	percentileIndex := int(float64(len(localEnergies)) * 0.25)
	if percentileIndex >= len(localEnergies) {
		percentileIndex = len(localEnergies) - 1
	}
	threshold := localEnergies[percentileIndex]

	// Apply bounds to prevent unreasonable values
	const maxThreshold = 0.08

	if threshold < v.config.MinAdaptiveEnergyThreshold {
		threshold = v.config.MinAdaptiveEnergyThreshold
	}
	if threshold > maxThreshold {
		threshold = maxThreshold
	}

	return threshold
}

// calculateDynamicZCRThreshold computes adaptive ZCR threshold from local window.
// Uses 60th percentile.
// Falls back to config value if threshold is outside reasonable bounds.
func (v *AdaptiveVAD) calculateDynamicZCRThreshold(zcrs []float64, index, windowSize int) float64 {
	// Calculate window bounds
	halfWindow := windowSize / 2
	start := index - halfWindow
	end := index + halfWindow + 1

	if start < 0 {
		start = 0
	}
	if end > len(zcrs) {
		end = len(zcrs)
	}

	// Extract local window
	localZCRs := make([]float64, end-start)
	copy(localZCRs, zcrs[start:end])

	// Sort to calculate percentile
	sort.Float64s(localZCRs)

	// Get 60th percentile
	percentileIndex := int(float64(len(localZCRs)) * 0.60)
	if percentileIndex >= len(localZCRs) {
		percentileIndex = len(localZCRs) - 1
	}
	threshold := localZCRs[percentileIndex]

	// Apply bounds checking - fall back to config if unreasonable
	const minThreshold = 0.01
	const maxThreshold = 0.5

	if threshold < minThreshold || threshold > maxThreshold {
		return v.config.ZCRThreshold
	}

	return threshold
}
