package vad

import (
	"math"
	"time"
)

// calculateEnergy computes the Root Mean Square (RMS) energy of a frame.
// Returns a value in range [0.0, 1.0] for normalized audio.
//
// Formula: RMS = sqrt(sum(x[i]²) / N)
func calculateEnergy(frame []float64) float64 {
	if len(frame) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, sample := range frame {
		sum += sample * sample
	}

	return math.Sqrt(sum / float64(len(frame)))
}

// calculateZCR computes the Zero Crossing Rate of a frame.
// Returns the fraction of samples where the signal crosses zero.
//
// Formula: ZCR = (number of sign changes) / (frame length)
func calculateZCR(frame []float64) float64 {
	if len(frame) <= 1 {
		return 0.0
	}

	crossings := 0
	for i := 1; i < len(frame); i++ {
		if (frame[i] >= 0 && frame[i-1] < 0) || (frame[i] < 0 && frame[i-1] >= 0) {
			crossings++
		}
	}

	return float64(crossings) / float64(len(frame))
}

// medianFilter applies median filtering to smooth binary speech detection results.
// Uses a sliding window to reduce isolated false positives/negatives.
func medianFilter(data []bool, windowSize int) []bool {
	if windowSize <= 0 || len(data) == 0 {
		return data
	}

	result := make([]bool, len(data))
	halfWindow := windowSize / 2

	for i := range data {
		// Calculate window bounds
		start := i - halfWindow
		end := i + halfWindow + 1

		if start < 0 {
			start = 0
		}
		if end > len(data) {
			end = len(data)
		}

		// Count true values in window
		trueCount := 0
		for j := start; j < end; j++ {
			if data[j] {
				trueCount++
			}
		}

		// Majority vote
		result[i] = trueCount > (end-start)/2
	}

	return result
}

// mergeSegments converts boolean speech indicators to speech segments.
// Filters segments by minimum duration and calculates average energy.
func mergeSegments(isSpeech []bool, audioData *AudioData, energies []float64, minDuration time.Duration) []SpeechSegment {
	if len(isSpeech) == 0 {
		return nil
	}

	frameDuration := float64(audioData.Duration) / float64(len(isSpeech))
	segments := []SpeechSegment{}

	var currentSegment *SpeechSegment
	energySum := 0.0
	frameCount := 0

	for i, speech := range isSpeech {
		if speech {
			if currentSegment == nil {
				// Start new segment
				currentSegment = &SpeechSegment{
					Start: time.Duration(float64(i) * frameDuration),
				}
				energySum = energies[i]
				frameCount = 1
			} else {
				// Continue current segment
				energySum += energies[i]
				frameCount++
			}
		} else {
			if currentSegment != nil {
				// End current segment
				currentSegment.End = time.Duration(float64(i) * frameDuration)
				currentSegment.Duration = currentSegment.End - currentSegment.Start
				currentSegment.Energy = energySum / float64(frameCount)

				// Only keep if meets minimum duration
				if currentSegment.Duration >= minDuration {
					segments = append(segments, *currentSegment)
				}

				currentSegment = nil
				energySum = 0.0
				frameCount = 0
			}
		}
	}

	// Handle open segment at end
	if currentSegment != nil {
		currentSegment.End = audioData.Duration
		currentSegment.Duration = currentSegment.End - currentSegment.Start
		currentSegment.Energy = energySum / float64(frameCount)

		if currentSegment.Duration >= minDuration {
			segments = append(segments, *currentSegment)
		}
	}

	return segments
}
