package main

import (
	"fmt"
	"os"

	"github.com/FlorinAndrei/soundspec-go/config"
	"github.com/FlorinAndrei/soundspec-go/process"
	"github.com/spf13/cobra"
)

func main() {
	cfg := config.DefaultConfig()

	rootCmd := &cobra.Command{
		Use:   "soundspec",
		Short: "Generate spectrograms from audio files",
		Long: `soundspec generates spectrogram images from audio files.

It uses ffmpeg for audio decoding (supports all common formats).
Spectrograms are rendered with a logarithmic frequency axis and
viridis colormap.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Input == "" {
				return fmt.Errorf("input is required (use -i or --input)")
			}
			processor := process.NewProcessor(cfg)
			return processor.Run()
		},
	}

	// Input/Output flags
	rootCmd.Flags().StringVarP(&cfg.Input, "input", "i", "", "Input file or directory (required)")
	rootCmd.Flags().StringVarP(&cfg.Output, "output", "o", cfg.Output, "Output directory")
	rootCmd.Flags().BoolVar(&cfg.Overwrite, "overwrite", cfg.Overwrite, "Overwrite existing output files")

	// Image flags
	rootCmd.Flags().IntVarP(&cfg.SpectrogramSize, "spectrogram-size", "s", cfg.SpectrogramSize, "Spectrogram area width in pixels")

	// Frequency flags
	rootCmd.Flags().IntVar(&cfg.FrequencyMin, "frequency-min", cfg.FrequencyMin, "Minimum frequency in Hz")
	rootCmd.Flags().IntVar(&cfg.FrequencyMax, "frequency-max", cfg.FrequencyMax, "Maximum frequency in Hz")
	rootCmd.Flags().IntVar(&cfg.FrequencyRes, "frequency-res", cfg.FrequencyRes, "Frequency resolution in Hz")

	// Processing flags
	rootCmd.Flags().IntVarP(&cfg.Workers, "workers", "w", cfg.Workers, "Number of parallel workers")

	// Other flags
	rootCmd.Flags().BoolVarP(&cfg.Quiet, "quiet", "q", cfg.Quiet, "Suppress output except errors")

	// Version flag
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("soundspec version {{.Version}}\n")

	// Mark input as required
	rootCmd.MarkFlagRequired("input")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
