package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Error types for distinguishing failure modes
var (
	ErrNoAudioStream = errors.New("no audio stream found")
	ErrNotAudioFile  = errors.New("not a recognized audio file")
)

// FFprobeError represents an error from ffprobe
type FFprobeError struct {
	Err error
}

func (e *FFprobeError) Error() string {
	return fmt.Sprintf("ffprobe: %v", e.Err)
}

func (e *FFprobeError) Unwrap() error {
	return e.Err
}

// FFmpegError represents an error from ffmpeg
type FFmpegError struct {
	Err error
}

func (e *FFmpegError) Error() string {
	return fmt.Sprintf("ffmpeg: %v", e.Err)
}

func (e *FFmpegError) Unwrap() error {
	return e.Err
}

// AudioInfo contains metadata about an audio file
type AudioInfo struct {
	SampleRate int
	Channels   int
}

// ProbeAudio checks if a file has an audio stream and returns its info.
// Returns ErrNoAudioStream if the file has no audio.
// Returns FFprobeError for ffprobe failures.
func ProbeAudio(path string) (*AudioInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=sample_rate,channels",
		"-of", "csv=p=0",
		path,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		// Check if ffprobe ran but returned non-zero (file not recognized)
		// vs couldn't run at all (binary not found, permissions, etc.)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// ffprobe ran but couldn't process the file - not a valid audio file
			return nil, ErrNotAudioFile
		}
		// ffprobe couldn't run at all - actual system error
		return nil, &FFprobeError{Err: fmt.Errorf("failed to run: %w", err)}
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, ErrNoAudioStream
	}

	parts := strings.Split(trimmed, ",")
	if len(parts) < 2 {
		return nil, &FFprobeError{Err: fmt.Errorf("unexpected output: %s", output)}
	}

	sampleRate, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, &FFprobeError{Err: fmt.Errorf("failed to parse sample rate: %w", err)}
	}

	channels, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, &FFprobeError{Err: fmt.Errorf("failed to parse channels: %w", err)}
	}

	return &AudioInfo{
		SampleRate: sampleRate,
		Channels:   channels,
	}, nil
}

// DecodeAudio decodes audio from a file using ffmpeg.
// The caller should have already verified the file has audio via ProbeAudio.
// Returns FFmpegError for ffmpeg failures.
func DecodeAudio(path string, info *AudioInfo) (*AudioData, error) {
	cmd := exec.Command("ffmpeg",
		"-i", path,
		"-f", "f32le",
		"-acodec", "pcm_f32le",
		"-v", "quiet",
		"pipe:1",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, &FFmpegError{Err: fmt.Errorf("failed: %w, stderr: %s", err, stderr.String())}
	}

	// Parse the raw PCM data
	data := stdout.Bytes()
	numSamples := len(data) / 4 // 4 bytes per float32
	samples := make([]float64, numSamples)

	reader := bytes.NewReader(data)
	for i := range numSamples {
		var sample float32
		if err := binary.Read(reader, binary.LittleEndian, &sample); err != nil {
			break
		}
		samples[i] = float64(sample)
	}

	// Convert to mono
	mono := convertToMono(samples, info.Channels)

	return &AudioData{
		Samples:    mono,
		SampleRate: info.SampleRate,
	}, nil
}

// ReadAudio probes and decodes an audio file.
// This is a convenience function that combines ProbeAudio and DecodeAudio.
func readWithFFmpeg(path string) (*AudioData, error) {
	info, err := ProbeAudio(path)
	if err != nil {
		return nil, err
	}
	return DecodeAudio(path, info)
}
