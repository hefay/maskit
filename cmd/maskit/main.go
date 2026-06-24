package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hefay/maskit"
)

const pollInterval = 2 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	key := flag.String("key", "", "API key (env MASKIT_API_KEY or ~/.maskit fallback)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	faces := flag.Bool("faces", true, "Detect and mask faces")
	humans := flag.Bool("humans", true, "Detect and mask humans")
	plates := flag.Bool("plates", true, "Detect and mask license plates")
	shape := flag.String("shape", "mask", "Mask shape: mask or rectangle")
	method := flag.String("method", "blur", "Masking method: blur or blackfill")
	blurStrength := flag.Int("blur-strength", 30, "Gaussian blur strength")
	edgeBlur := flag.Float64("edge-blur", 0.2, "Edge blur size (0.0–1.0)")
	output := flag.String("output", "", "Output file path")

	flag.BoolVar(verbose, "v", false, "Verbose output (short)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: maskit [flags] <image-file>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	inputPath := flag.Arg(0)

	apiKey := resolveAPIKey(*key)
	if apiKey == "" {
		return errors.New("API key required: provide via --key, MASKIT_API_KEY env, or ~/.maskit file")
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open image: %w", err)
	}
	defer file.Close()

	ctx := context.Background()
	service := maskit.NewMaskingService(maskit.WithApiKey(apiKey))

	opts := []maskit.MaskOption{
		maskit.WithFaces(*faces),
		maskit.WithHumans(*humans),
		maskit.WithLicensePlates(*plates),
		maskit.WithBlurStrength(*blurStrength),
		maskit.WithEdgeBlurSize(float32(*edgeBlur)),
	}

	switch strings.ToLower(*shape) {
	case "mask":
		opts = append(opts, maskit.WithShape(maskit.ShapeMask))
	case "rectangle":
		opts = append(opts, maskit.WithShape(maskit.ShapeRectangle))
	default:
		return fmt.Errorf("invalid shape %q: use mask or rectangle", *shape)
	}

	switch strings.ToLower(*method) {
	case "blur":
		opts = append(opts, maskit.WithMethod(maskit.MethodBlur))
	case "blackfill":
		opts = append(opts, maskit.WithMethod(maskit.MethodBlackFill))
	default:
		return fmt.Errorf("invalid method %q: use blur or blackfill", *method)
	}

	req := maskit.PrepareForMasking(file)
	for _, opt := range opts {
		opt(&req)
	}

	log(*verbose, "Submitting image...")

	resp, err := service.RequestMasking(ctx, req)
	if err != nil {
		return wrapError(fmt.Errorf("request failed: %w", err))
	}

	verboseLogf(*verbose, "Job submitted: %s", resp.JobID)
	verboseLogf(*verbose, "Waiting for processing...")

	start := time.Now()
	const maxDots = 60

	for {
		status, err := service.GetJobStatus(ctx, resp.JobID)
		if err != nil {
			fmt.Println()
			return wrapError(fmt.Errorf("status check failed: %w", err))
		}

		switch status.Status {
		case maskit.JobStatusReadyToDownload, maskit.JobStatusCompleted:
			elapsed := time.Since(start).Truncate(time.Second)
			verboseLogf(*verbose, "Ready after %s", elapsed)
			if !*verbose {
				fmt.Println(" done!")
			}

			log(*verbose, "Downloading masked image...")

			reader, err := service.DownloadImage(ctx, resp.JobID)
			if err != nil {
				return wrapError(fmt.Errorf("download failed: %w", err))
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("cannot read response: %w", err)
			}

			outPath := *output
			if outPath == "" {
				ext := filepath.Ext(inputPath)
				base := strings.TrimSuffix(inputPath, ext)
				outPath = base + "_masked" + ext
			}

			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("cannot write output: %w", err)
			}

			log(*verbose, "Saved to "+outPath)
			return nil

		case maskit.JobStatusFailed:
			fmt.Println()
			return fmt.Errorf("job %s failed", resp.JobID)

		case maskit.JobStatusTimedOut:
			fmt.Println()
			return fmt.Errorf("job %s timed out", resp.JobID)

		default:
			if *verbose {
				elapsed := time.Since(start).Truncate(time.Second)
				fmt.Printf("  [%4s] Status: %s\n", elapsed, status.Status)
			} else {
				fmt.Print(".")
				// break line if too many dots to avoid runaway lines
				if dots := int(time.Since(start)/pollInterval) % maxDots; dots == 0 {
					fmt.Printf(" %s\n", time.Since(start).Truncate(time.Second))
				}
			}
		}

		time.Sleep(pollInterval)
	}
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "401") {
		return fmt.Errorf("%w\n\nHint: Check that your API key is correct. Use --key, MASKIT_API_KEY env, or ~/.maskit", err)
	}
	return err
}

func log(verbose bool, msg string) {
	if verbose {
		fmt.Println(msg)
	}
}

func verboseLogf(verbose bool, format string, args ...any) {
	if verbose {
		fmt.Printf(format+"\n", args...)
	}
}

func resolveAPIKey(flagKey string) string {
	if flagKey != "" {
		return flagKey
	}

	if env := os.Getenv("MASKIT_API_KEY"); env != "" {
		return env
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".maskit"))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
}
