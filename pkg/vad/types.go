package vad

import "time"

// AudioData represents loaded audio with normalized samples.
type AudioData struct {
	// Samples contains PCM samples normalized to [-1.0, 1.0]
	Samples []float64

	// SampleRate is the sample rate in Hz (e.g., 16000, 44100)
	SampleRate int

	// Channels is the number of channels (1=mono, 2=stereo)
	Channels int

	// Duration is the total duration of audio
	Duration time.Duration

	// FileName is the original filename
	FileName string
}

// SpeechSegment represents a detected speech region.
type SpeechSegment struct {
	// Start time of speech segment
	Start time.Duration

	// End time of speech segment
	End time.Duration

	// Speaker label (e.g., "agent" or "user")
	Speaker string

	// Energy is the average RMS energy of segment
	Energy float64

	// Duration of segment (End - Start)
	Duration time.Duration
}

// GetMonoSamples returns mono samples, averaging channels if stereo.
func (ad *AudioData) GetMonoSamples() []float64 {
	if ad.Channels == 1 {
		return ad.Samples
	}

	// Average stereo channels to mono
	monoLength := len(ad.Samples) / ad.Channels
	mono := make([]float64, monoLength)

	for i := 0; i < monoLength; i++ {
		sum := 0.0
		for ch := 0; ch < ad.Channels; ch++ {
			sum += ad.Samples[i*ad.Channels+ch]
		}
		mono[i] = sum / float64(ad.Channels)
	}

	return mono
}

// GetFrames extracts overlapping frames from the audio.
// frameSize is the duration of each frame (e.g., 25ms)
// hopSize is the stride between frames (e.g., 10ms)
func (ad *AudioData) GetFrames(frameSize, hopSize time.Duration) [][]float64 {
	mono := ad.GetMonoSamples()

	frameSamples := int(float64(ad.SampleRate) * frameSize.Seconds())
	hopSamples := int(float64(ad.SampleRate) * hopSize.Seconds())

	if frameSamples <= 0 || hopSamples <= 0 || len(mono) < frameSamples {
		return [][]float64{}
	}

	// Pre-allocate frames array
	numFrames := (len(mono)-frameSamples)/hopSamples + 1
	frames := make([][]float64, 0, numFrames)

	for position := 0; position+frameSamples <= len(mono); position += hopSamples {
		frame := mono[position : position+frameSamples]
		frames = append(frames, frame)
	}

	return frames
}
