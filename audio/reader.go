package audio

// AudioData holds the decoded audio information
type AudioData struct {
	Samples    []float64 // Mono audio samples normalized to [-1, 1]
	SampleRate int
}

// ReadAudio reads an audio file using ffmpeg
func ReadAudio(path string) (*AudioData, error) {
	return readWithFFmpeg(path)
}

// convertToMono converts multi-channel audio to mono by averaging channels
func convertToMono(samples []float64, channels int) []float64 {
	if channels == 1 {
		return samples
	}

	numFrames := len(samples) / channels
	mono := make([]float64, numFrames)

	for i := range numFrames {
		sum := 0.0
		for ch := range channels {
			sum += samples[i*channels+ch]
		}
		mono[i] = sum / float64(channels)
	}

	return mono
}
