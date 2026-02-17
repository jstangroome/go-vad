package vad

import "time"

// StreamEventType represents the type of streaming VAD event.
type StreamEventType int

const (
	// EventNone indicates no event (silence continuing or speech continuing).
	EventNone StreamEventType = iota

	// EventSpeechStarted indicates speech activity has begun.
	EventSpeechStarted

	// EventSpeechEnded indicates speech activity has ended.
	EventSpeechEnded
)

// String returns the string representation of the event type.
func (e StreamEventType) String() string {
	switch e {
	case EventNone:
		return "None"
	case EventSpeechStarted:
		return "SpeechStarted"
	case EventSpeechEnded:
		return "SpeechEnded"
	default:
		return "Unknown"
	}
}

// StreamEvent represents a real-time VAD event during streaming audio processing.
// Events are emitted when speech state transitions occur (silence to speech or vice versa).
type StreamEvent struct {
	// Type of event (EventNone, EventSpeechStarted, or EventSpeechEnded).
	Type StreamEventType

	// Timestamp of the event relative to the start of the audio stream.
	Timestamp time.Duration

	// Segment contains the complete speech segment details when Type is EventSpeechEnded.
	// This is nil for EventSpeechStarted and EventNone.
	Segment *SpeechSegment
}
