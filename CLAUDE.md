# soundspec-go

Audio spectrogram generator written in Go. Reads audio files and generates PNG spectrograms with logarithmic frequency axis and viridis colormap.

## Build

```bash
make dev      # builds for current platform to build/soundspec
make          # builds for all platforms (macOS/Linux/Windows × AMD64/ARM64)
make clean    # removes build directory
```

Cross-platform binaries are named `soundspec-{os}-{arch}` (e.g., `soundspec-linux-amd64`, `soundspec-windows-arm64.exe`).

## Usage

```bash
# Single file
./soundspec -i audio.mp3 -o /output/dir

# Recursive directory (processes all files, skips non-audio)
./soundspec -i /music/folder -o /spectrograms --workers 8
```

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-i, --input` | Input file or directory (required) | - |
| `-o, --output` | Output directory | `.` |
| `-s, --spectrogram-size` | Spectrogram area width in pixels | 1000 |
| `--frequency-min` | Minimum frequency (Hz) | 10 |
| `--frequency-max` | Maximum frequency (Hz) | 20000 |
| `--frequency-res` | Frequency resolution (Hz) | 1 |
| `-w, --workers` | Parallel workers | CPU cores |
| `--overwrite` | Overwrite existing output files | false |
| `-q, --quiet` | Suppress non-error output | false |
| `-v, --version` | Show version | - |

## Project Structure

```
├── main.go                 # CLI entry point (cobra)
├── version.go              # Build-time version injection
├── config/
│   └── config.go           # Config struct, defaults, margins, aspect ratio
├── audio/
│   ├── reader.go           # AudioData struct, ReadAudio
│   └── ffmpeg.go           # FFmpeg/ffprobe integration, typed errors
├── dsp/
│   └── spectrogram.go      # STFT, Hann window, log-frequency resampling
├── render/
│   ├── colormap.go         # Viridis colormap (via dim13/colormap)
│   └── plot.go             # Image rendering with gg library
└── process/
    └── processor.go        # File discovery, worker pool, job orchestration
```

## Key Design Decisions

- **Spectrogram size**: `-s` specifies the spectrogram AREA dimensions; margins are added for axes/labels
- **Aspect ratio**: Fixed at 16:9 (defined in config/config.go)
- **Frequency axis**: Logarithmic scale, data is pre-computed in log-frequency bins
- **Color mapping**: Min/max spectrogram values map directly to min/max viridis colors (no color bar)
- **Audio detection**: ffprobe determines if a file has audio (no extension-based filtering)
- **Audio decoding**: ffmpeg for all formats (supports any format ffmpeg supports)
- **Output naming**: `input.mp3` → `input.mp3.png` (extension appended, not replaced)
- **Result categories**: Non-audio files are skipped; errors tracked separately for ffprobe (system issues), ffmpeg (decode failures), and internal (DSP/render)

## Dependencies

- `github.com/spf13/cobra` - CLI
- `gonum.org/v1/gonum/dsp/fourier` - FFT
- `git.sr.ht/~sbinet/gg` - 2D graphics rendering
- `github.com/golang/freetype` - TrueType font parsing
- `golang.org/x/image/font/gofont/goregular` - Embedded TrueType font
- `github.com/dim13/colormap` - Viridis colormap

External: `ffmpeg` and `ffprobe` must be installed and available in PATH.

## Release

GitHub Actions workflow (`.github/workflows/release.yml`) builds and releases binaries:
- Triggered manually via `workflow_dispatch`
- Builds all 6 platform binaries using `make all`
- Creates a GitHub release with version tag from build output (format: `YYYYMMDD-HHMMSS`)

## Constants Location

All configurable constants are in `config/config.go`:
- `AspectRatio` (16:9)
- `MarginLeft`, `MarginRight`, `MarginTop`, `MarginBottom`
- Default values for all CLI flags
