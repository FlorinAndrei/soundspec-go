package process

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/FlorinAndrei/soundspec-go/audio"
	"github.com/FlorinAndrei/soundspec-go/config"
	"github.com/FlorinAndrei/soundspec-go/dsp"
	"github.com/FlorinAndrei/soundspec-go/render"
)

// ResultType represents the outcome of processing a file
type ResultType int

const (
	ResultSuccess ResultType = iota
	ResultSkipNoAudio
	ResultSkipExists
	ResultErrorFFprobe
	ResultErrorFFmpeg
	ResultErrorInternal
)

// Job represents a single file processing job
type Job struct {
	InputPath  string
	OutputPath string
}

// Result represents the result of processing a job
type Result struct {
	Job  Job
	Type ResultType
	Err  error
}

// Stats holds processing statistics
type Stats struct {
	Success       int
	SkipNoAudio   int
	SkipExists    int
	ErrorFFprobe  int
	ErrorFFmpeg   int
	ErrorInternal int
}

// Processor handles file discovery and processing
type Processor struct {
	cfg config.Config
}

// NewProcessor creates a new Processor
func NewProcessor(cfg config.Config) *Processor {
	return &Processor{cfg: cfg}
}

// log logs a message if not in quiet mode
func (p *Processor) log(format string, args ...interface{}) {
	if !p.cfg.Quiet {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// logError logs an error (always, even in quiet mode)
func (p *Processor) logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
}

// Run processes all files based on configuration
func (p *Processor) Run() error {
	jobs, skippedExists, err := p.discoverJobs()
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	// Log skipped files (output exists)
	for _, job := range skippedExists {
		p.log("Skipping %s (output exists)", job.InputPath)
	}

	if len(jobs) == 0 && len(skippedExists) == 0 {
		p.log("No files found")
		return nil
	}

	if len(jobs) == 0 {
		p.log("No files to process (all outputs exist)")
		p.printSummary(Stats{SkipExists: len(skippedExists)})
		return nil
	}

	p.log("Found %d file(s) to process", len(jobs))

	// Process jobs using worker pool
	results := p.processJobs(jobs)

	// Collect statistics
	stats := Stats{SkipExists: len(skippedExists)}
	for _, result := range results {
		switch result.Type {
		case ResultSuccess:
			stats.Success++
		case ResultSkipNoAudio:
			stats.SkipNoAudio++
		case ResultSkipExists:
			stats.SkipExists++
		case ResultErrorFFprobe:
			stats.ErrorFFprobe++
		case ResultErrorFFmpeg:
			stats.ErrorFFmpeg++
		case ResultErrorInternal:
			stats.ErrorInternal++
		}
	}

	p.printSummary(stats)

	totalErrors := stats.ErrorFFprobe + stats.ErrorFFmpeg + stats.ErrorInternal
	if totalErrors > 0 && stats.Success == 0 {
		return fmt.Errorf("all files failed to process")
	}

	return nil
}

// printSummary prints the processing summary
func (p *Processor) printSummary(stats Stats) {
	var parts []string

	if stats.Success > 0 {
		parts = append(parts, fmt.Sprintf("%d spectrograms created", stats.Success))
	}
	if stats.SkipNoAudio > 0 {
		parts = append(parts, fmt.Sprintf("%d non-audio skipped", stats.SkipNoAudio))
	}
	if stats.SkipExists > 0 {
		parts = append(parts, fmt.Sprintf("%d existing skipped", stats.SkipExists))
	}
	if stats.ErrorFFprobe > 0 {
		parts = append(parts, fmt.Sprintf("%d ffprobe errors", stats.ErrorFFprobe))
	}
	if stats.ErrorFFmpeg > 0 {
		parts = append(parts, fmt.Sprintf("%d ffmpeg errors", stats.ErrorFFmpeg))
	}
	if stats.ErrorInternal > 0 {
		parts = append(parts, fmt.Sprintf("%d internal errors", stats.ErrorInternal))
	}

	if len(parts) > 0 {
		fmt.Fprintf(os.Stderr, "Summary: %s\n", joinParts(parts))
	}
}

// joinParts joins parts with commas
func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// discoverJobs finds all files to process and creates jobs
// Returns jobs to process and jobs skipped due to existing output
func (p *Processor) discoverJobs() ([]Job, []Job, error) {
	inputInfo, err := os.Stat(p.cfg.Input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat input: %w", err)
	}

	var jobs []Job

	if inputInfo.IsDir() {
		// Recursive mode - collect all files
		jobs, err = p.discoverRecursive()
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Single file mode
		outputPath := p.singleFileOutputPath(p.cfg.Input)
		jobs = []Job{{InputPath: p.cfg.Input, OutputPath: outputPath}}
	}

	// Filter out existing files unless overwrite is enabled
	if !p.cfg.Overwrite {
		var toProcess, skipped []Job
		for _, job := range jobs {
			if _, err := os.Stat(job.OutputPath); os.IsNotExist(err) {
				toProcess = append(toProcess, job)
			} else {
				skipped = append(skipped, job)
			}
		}
		return toProcess, skipped, nil
	}

	return jobs, nil, nil
}

// discoverRecursive walks the input directory and finds all files
func (p *Processor) discoverRecursive() ([]Job, error) {
	var jobs []Job

	inputAbs, err := filepath.Abs(p.cfg.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Include the input directory name in output path
	inputBaseName := filepath.Base(inputAbs)

	err = filepath.WalkDir(inputAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files/dirs we can't access
		}

		if d.IsDir() {
			return nil
		}

		// Calculate relative path from input directory
		relPath, err := filepath.Rel(inputAbs, path)
		if err != nil {
			return nil
		}

		// Construct output path (include input directory name)
		outputPath := filepath.Join(p.cfg.Output, inputBaseName, relPath+".png")

		jobs = append(jobs, Job{InputPath: path, OutputPath: outputPath})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return jobs, nil
}

// singleFileOutputPath returns the output path for single file mode
func (p *Processor) singleFileOutputPath(inputPath string) string {
	basename := filepath.Base(inputPath)
	return filepath.Join(p.cfg.Output, basename+".png")
}

// processJobs processes jobs using a worker pool
func (p *Processor) processJobs(jobs []Job) []Result {
	numWorkers := p.cfg.Workers
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	jobsChan := make(chan Job, len(jobs))
	resultsChan := make(chan Result, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				result := p.processFile(job)
				resultsChan <- result
			}
		}()
	}

	// Send jobs
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultsChan)

	// Collect results
	var results []Result
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// processFile processes a single file
func (p *Processor) processFile(job Job) Result {
	// Probe the file for audio
	info, err := audio.ProbeAudio(job.InputPath)
	if err != nil {
		if errors.Is(err, audio.ErrNoAudioStream) {
			p.log("Skipping %s (no audio stream)", job.InputPath)
			return Result{Job: job, Type: ResultSkipNoAudio}
		}
		if errors.Is(err, audio.ErrNotAudioFile) {
			p.log("Skipping %s (not a recognized audio file)", job.InputPath)
			return Result{Job: job, Type: ResultSkipNoAudio}
		}
		// FFprobe system error (binary not found, permissions, etc.)
		p.logError("%s: %v", job.InputPath, err)
		return Result{Job: job, Type: ResultErrorFFprobe, Err: err}
	}

	// Decode audio
	audioData, err := audio.DecodeAudio(job.InputPath, info)
	if err != nil {
		p.logError("%s: %v", job.InputPath, err)
		return Result{Job: job, Type: ResultErrorFFmpeg, Err: err}
	}

	// Check sample rate constraints
	effectiveFreqMax := p.cfg.FrequencyMax
	nyquist := audioData.SampleRate / 2

	if nyquist < p.cfg.FrequencyMin {
		err := fmt.Errorf("sample rate %d Hz is too low (Nyquist %d Hz < minimum frequency %d Hz)",
			audioData.SampleRate, nyquist, p.cfg.FrequencyMin)
		p.logError("%s: %v", job.InputPath, err)
		return Result{Job: job, Type: ResultErrorInternal, Err: err}
	}

	if nyquist < effectiveFreqMax {
		p.log("Warning: %s - sample rate %d Hz limits maximum frequency to %d Hz",
			job.InputPath, audioData.SampleRate, nyquist)
		effectiveFreqMax = nyquist
	}

	// Compute spectrogram
	result := dsp.ComputeSpectrogram(
		audioData.Samples,
		audioData.SampleRate,
		p.cfg.FrequencyMin,
		effectiveFreqMax,
		p.cfg.FrequencyRes,
		p.cfg.SpectrogramSize,
		p.cfg.SpectrogramHeight(),
	)

	// Ensure output directory exists
	outputDir := filepath.Dir(job.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		err = fmt.Errorf("failed to create output directory: %w", err)
		p.logError("%s: %v", job.InputPath, err)
		return Result{Job: job, Type: ResultErrorInternal, Err: err}
	}

	// Render spectrogram
	title := filepath.Base(job.InputPath)
	if err := render.RenderSpectrogram(result, title, job.OutputPath, p.cfg); err != nil {
		err = fmt.Errorf("failed to render spectrogram: %w", err)
		p.logError("%s: %v", job.InputPath, err)
		return Result{Job: job, Type: ResultErrorInternal, Err: err}
	}

	p.log("Created: %s", job.OutputPath)
	return Result{Job: job, Type: ResultSuccess}
}
