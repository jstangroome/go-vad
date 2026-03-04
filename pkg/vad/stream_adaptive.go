package vad

import (
	"sort"
	"time"
)

// StreamingAdaptiveVAD implements Voice Activity Detection for streaming audio
// with adaptive thresholds. Dynamically adjusts thresholds based on recent audio characteristics.
type StreamingAdaptiveVAD struct {
	config *Config

	// Processing state
	sampleBuffer []float64 // Incomplete frame samples
	totalSamples int64     // Total samples processed (for timestamp calculation)
	sampleRate   int       // Required for time calculations

	// Speech state tracking
	inSpeech        bool          // Currently in speech segment
	speechStartTime time.Duration // When current segment started
	speechEnergy    float64       // Accumulated energy for segment
	speechFrames    int           // Number of frames in segment

	// Median filter state (window size 5 for adaptive)
	recentDecisions []bool // Last 5 frame decisions

	// Feature history for adaptive thresholds
	energyHistory  []float64 // Rolling window of last 100 energies
	zcrHistory     []float64 // Rolling window of last 100 ZCRs
	historyMaxSize int       // Maximum history size (100 frames)

	// Frame processing
	frameSize int // Frame size in samples
	hopSize   int // Hop size in samples
}

// NewStreamingAdaptiveVAD creates a new adaptive streaming VAD instance.
// Requires a sample rate to calculate frame sizes and timestamps.
// If config is nil, uses default configuration.
func NewStreamingAdaptiveVAD(config *Config, sampleRate int) *StreamingAdaptiveVAD {
	if config == nil {
		config = DefaultConfig()
	}

	// Calculate frame and hop sizes in samples
	frameSize := int(float64(sampleRate) * config.FrameSize.Seconds())
	hopSize := int(float64(sampleRate) * config.HopSize.Seconds())

	return &StreamingAdaptiveVAD{
		config:          config,
		sampleRate:      sampleRate,
		sampleBuffer:    make([]float64, 0, frameSize),
		frameSize:       frameSize,
		hopSize:         hopSize,
		recentDecisions: make([]bool, 0, 5),
		energyHistory:   make([]float64, 0, 100),
		zcrHistory:      make([]float64, 0, 100),
		historyMaxSize:  100,
		inSpeech:        false,
	}
}

func (s *StreamingAdaptiveVAD) ResetFeatureHistoryCapacity(newCapacityInHops int) {
	s.historyMaxSize = newCapacityInHops
	s.energyHistory = make([]float64, 0, newCapacityInHops)
	s.zcrHistory = make([]float64, 0, newCapacityInHops)
}

// ProcessChunk processes a chunk of audio samples and returns events.
// Samples should be mono audio normalized to [-1.0, 1.0].
// Returns a StreamEvent which may be EventNone, EventSpeechStarted, or EventSpeechEnded.
func (s *StreamingAdaptiveVAD) ProcessChunk(samples []float64) StreamEvent {
	// Append samples to buffer
	s.sampleBuffer = append(s.sampleBuffer, samples...)

	var event StreamEvent
	event.Type = EventNone

	// Process all complete frames
	for len(s.sampleBuffer) >= s.frameSize {
		// Extract frame
		frame := s.sampleBuffer[:s.frameSize]

		// Process frame and check for events
		frameEvent := s.processFrame(frame)

		// Keep the most recent non-None event
		if frameEvent.Type != EventNone {
			event = frameEvent
		}

		// Slide buffer by hopSize
		copy(s.sampleBuffer, s.sampleBuffer[s.hopSize:])
		s.sampleBuffer = s.sampleBuffer[:len(s.sampleBuffer)-s.hopSize]
		s.totalSamples += int64(s.hopSize)
	}

	return event
}

// processFrame processes a single frame and returns event.
func (s *StreamingAdaptiveVAD) processFrame(frame []float64) StreamEvent {
	// Calculate features using existing functions
	energy := calculateEnergy(frame)
	zcr := calculateZCR(frame)

	// Update feature history
	s.updateFeatureHistory(energy, zcr)

	// Calculate dynamic thresholds from history
	energyThreshold := s.calculateDynamicThresholdStreaming()
	zcrThreshold := s.calculateDynamicZCRThresholdStreaming()

	// Apply threshold (dual-threshold decision)
	isSpeech := energy > energyThreshold && zcr < zcrThreshold

	// Apply median filter to smooth decisions
	isSpeech = s.applyMedianFilter(isSpeech)

	// Update state machine and generate events
	return s.updateState(isSpeech, energy)
}

// updateFeatureHistory maintains circular buffers for energy and ZCR.
func (s *StreamingAdaptiveVAD) updateFeatureHistory(energy, zcr float64) {
	// Add to energy history
	s.energyHistory = append(s.energyHistory, energy)
	if len(s.energyHistory) > s.historyMaxSize {
		s.energyHistory = s.energyHistory[1:]
	}

	// Add to ZCR history
	s.zcrHistory = append(s.zcrHistory, zcr)
	if len(s.zcrHistory) > s.historyMaxSize {
		s.zcrHistory = s.zcrHistory[1:]
	}
}

// calculateDynamicThresholdStreaming computes adaptive energy threshold from history.
// Uses 25th percentile (top 75% considered potential speech).
// Falls back to config threshold if insufficient history.
func (s *StreamingAdaptiveVAD) calculateDynamicThresholdStreaming() float64 {
	// If insufficient history, use config threshold
	if len(s.energyHistory) < 20 {
		return s.config.EnergyThreshold
	}

	// Copy history for sorting (don't modify original)
	localEnergies := make([]float64, len(s.energyHistory))
	copy(localEnergies, s.energyHistory)

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

	if threshold < s.config.MinAdaptiveEnergyThreshold {
		threshold = s.config.MinAdaptiveEnergyThreshold
	}
	if threshold > maxThreshold {
		threshold = maxThreshold
	}

	return threshold
}

// calculateDynamicZCRThresholdStreaming computes adaptive ZCR threshold from history.
// Uses 60th percentile.
// Falls back to config threshold if insufficient history or unreasonable value.
func (s *StreamingAdaptiveVAD) calculateDynamicZCRThresholdStreaming() float64 {
	// If insufficient history, use config threshold
	if len(s.zcrHistory) < 20 {
		return s.config.ZCRThreshold
	}

	// Copy history for sorting (don't modify original)
	localZCRs := make([]float64, len(s.zcrHistory))
	copy(localZCRs, s.zcrHistory)

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
		return s.config.ZCRThreshold
	}

	return threshold
}

// applyMedianFilter applies median filtering using recent decisions.
// Maintains a circular buffer of the last 5 decisions (larger window for adaptive).
func (s *StreamingAdaptiveVAD) applyMedianFilter(decision bool) bool {
	s.recentDecisions = append(s.recentDecisions, decision)

	// Keep only last 5 decisions
	if len(s.recentDecisions) > 5 {
		s.recentDecisions = s.recentDecisions[1:]
	}

	// If fewer than 5 decisions, return current decision
	if len(s.recentDecisions) < 5 {
		return decision
	}

	// Count true values for majority vote
	trueCount := 0
	for _, d := range s.recentDecisions {
		if d {
			trueCount++
		}
	}

	// Majority vote
	return trueCount >= 3
}

// updateState manages speech state transitions and generates events.
func (s *StreamingAdaptiveVAD) updateState(isSpeech bool, energy float64) StreamEvent {
	currentTime := time.Duration(float64(s.totalSamples) / float64(s.sampleRate) * float64(time.Second))

	var event StreamEvent
	event.Type = EventNone
	event.Timestamp = currentTime

	if isSpeech && !s.inSpeech {
		// Transition: silence → speech
		s.inSpeech = true
		s.speechStartTime = currentTime
		s.speechEnergy = energy
		s.speechFrames = 1

		event.Type = EventSpeechStarted

	} else if !isSpeech && s.inSpeech {
		// Transition: speech → silence
		s.inSpeech = false

		// Calculate segment duration
		duration := currentTime - s.speechStartTime

		// Only emit if meets minimum duration
		if duration >= s.config.MinSpeechDuration {
			event.Type = EventSpeechEnded
			event.Segment = &SpeechSegment{
				Start:    s.speechStartTime,
				End:      currentTime,
				Duration: duration,
				Energy:   s.speechEnergy / float64(s.speechFrames),
			}
		}

		s.speechEnergy = 0
		s.speechFrames = 0

	} else if isSpeech && s.inSpeech {
		// Continue speech: accumulate energy
		s.speechEnergy += energy
		s.speechFrames++
	}

	return event
}

// Flush processes any remaining buffered samples and finalizes the stream.
// If there is an active speech segment, it will be emitted if it meets minimum duration.
// Call this when the audio stream ends to ensure all data is processed.
func (s *StreamingAdaptiveVAD) Flush() StreamEvent {
	var event StreamEvent
	event.Type = EventNone

	// If we have a partial frame, pad with zeros and process
	if len(s.sampleBuffer) > 0 {
		// Pad to frame size
		padding := make([]float64, s.frameSize-len(s.sampleBuffer))
		fullFrame := append(s.sampleBuffer, padding...)

		event = s.processFrame(fullFrame)
		s.sampleBuffer = s.sampleBuffer[:0]
	}

	// If still in speech at end, emit final segment
	if s.inSpeech {
		currentTime := time.Duration(float64(s.totalSamples) / float64(s.sampleRate) * float64(time.Second))
		duration := currentTime - s.speechStartTime

		if duration >= s.config.MinSpeechDuration {
			event.Type = EventSpeechEnded
			event.Timestamp = currentTime
			event.Segment = &SpeechSegment{
				Start:    s.speechStartTime,
				End:      currentTime,
				Duration: duration,
				Energy:   s.speechEnergy / float64(s.speechFrames),
			}
		}

		s.inSpeech = false
	}

	return event
}

// Reset clears all internal state, allowing the StreamingAdaptiveVAD to be reused.
// Call this to start processing a new audio stream.
func (s *StreamingAdaptiveVAD) Reset() {
	s.sampleBuffer = s.sampleBuffer[:0]
	s.totalSamples = 0
	s.inSpeech = false
	s.speechStartTime = 0
	s.speechEnergy = 0
	s.speechFrames = 0
	s.recentDecisions = s.recentDecisions[:0]
	s.energyHistory = s.energyHistory[:0]
	s.zcrHistory = s.zcrHistory[:0]
}
