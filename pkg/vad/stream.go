package vad

import "time"

// StreamingVAD implements Voice Activity Detection for streaming audio.
// It processes audio incrementally in chunks and emits events when speech starts/ends.
type StreamingVAD struct {
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

	// Median filter state (window size 3)
	recentDecisions []bool // Last 3 frame decisions

	// Frame processing
	frameSize int // Frame size in samples
	hopSize   int // Hop size in samples
}

// NewStreamingVAD creates a new streaming VAD instance.
// Requires a sample rate to calculate frame sizes and timestamps.
// If config is nil, uses default configuration.
func NewStreamingVAD(config *Config, sampleRate int) *StreamingVAD {
	if config == nil {
		config = DefaultConfig()
	}

	// Calculate frame and hop sizes in samples
	frameSize := int(float64(sampleRate) * config.FrameSize.Seconds())
	hopSize := int(float64(sampleRate) * config.HopSize.Seconds())

	return &StreamingVAD{
		config:          config,
		sampleRate:      sampleRate,
		sampleBuffer:    make([]float64, 0, frameSize),
		frameSize:       frameSize,
		hopSize:         hopSize,
		recentDecisions: make([]bool, 0, 3),
		inSpeech:        false,
	}
}

// ProcessChunk processes a chunk of audio samples and returns events.
// Samples should be mono audio normalized to [-1.0, 1.0].
// Returns a StreamEvent which may be EventNone, EventSpeechStarted, or EventSpeechEnded.
func (s *StreamingVAD) ProcessChunk(samples []float64) StreamEvent {
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
func (s *StreamingVAD) processFrame(frame []float64) StreamEvent {
	// Calculate features using existing functions
	energy := calculateEnergy(frame)
	var zcr float64 = -1
	if !s.config.DisableZCR {
		zcr = calculateZCR(frame)
	}

	// Apply threshold (dual-threshold decision)
	isSpeech := energy > s.config.EnergyThreshold &&
		zcr < s.config.ZCRThreshold

	// Apply median filter to smooth decisions
	isSpeech = s.applyMedianFilter(isSpeech)

	// Update state machine and generate events
	return s.updateState(isSpeech, energy)
}

// applyMedianFilter applies median filtering using recent decisions.
// Maintains a circular buffer of the last 3 decisions.
func (s *StreamingVAD) applyMedianFilter(decision bool) bool {
	s.recentDecisions = append(s.recentDecisions, decision)

	// Keep only last 3 decisions
	if len(s.recentDecisions) > 3 {
		s.recentDecisions = s.recentDecisions[1:]
	}

	// If fewer than 3 decisions, return current decision
	if len(s.recentDecisions) < 3 {
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
	return trueCount >= 2
}

// updateState manages speech state transitions and generates events.
func (s *StreamingVAD) updateState(isSpeech bool, energy float64) StreamEvent {
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
func (s *StreamingVAD) Flush() StreamEvent {
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

// Reset clears all internal state, allowing the StreamingVAD to be reused.
// Call this to start processing a new audio stream.
func (s *StreamingVAD) Reset() {
	s.sampleBuffer = s.sampleBuffer[:0]
	s.totalSamples = 0
	s.inSpeech = false
	s.speechStartTime = 0
	s.speechEnergy = 0
	s.speechFrames = 0
	s.recentDecisions = s.recentDecisions[:0]
}
