package vad

// VAD implements basic Voice Activity Detection using fixed thresholds.
// Uses dual-feature approach: RMS energy and Zero Crossing Rate (ZCR).
type VAD struct {
	config *Config
}

// NewVAD creates a new basic VAD instance with the given configuration.
// If config is nil, uses default configuration.
func NewVAD(config *Config) *VAD {
	if config == nil {
		config = DefaultConfig()
	}
	return &VAD{config: config}
}

// DetectSpeech detects speech segments in audio using fixed thresholds.
//
// Algorithm:
// 1. Extract frames from audio (overlapping windows)
// 2. Calculate energy (RMS) and ZCR for each frame
// 3. Apply thresholds: speech if (energy > threshold) AND (zcr < threshold)
// 4. Apply median filter to smooth results
// 5. Merge consecutive speech frames into segments
// 6. Filter by minimum duration
//
// Returns array of detected speech segments.
func (v *VAD) DetectSpeech(audioData *AudioData) []SpeechSegment {
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
		zcrs[i] = calculateZCR(frame)

		// Dual-threshold decision: high energy AND low ZCR indicates speech
		isSpeech[i] = energies[i] > v.config.EnergyThreshold &&
			zcrs[i] < v.config.ZCRThreshold
	}

	// Apply median filter to smooth detection (window size 3)
	isSpeech = medianFilter(isSpeech, 3)

	// Merge consecutive speech frames into segments
	segments := mergeSegments(isSpeech, audioData, energies, v.config.MinSpeechDuration)

	return segments
}
