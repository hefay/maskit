package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hefay/maskit"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	key := flag.String("key", "", "API key (env MASKIT_API_KEY or ~/.maskit fallback)")
	faces := flag.Bool("faces", true, "Detect and mask faces")
	humans := flag.Bool("humans", true, "Detect and mask humans")
	plates := flag.Bool("plates", true, "Detect and mask license plates")
	shape := flag.String("shape", "mask", "Mask shape: mask or rectangle")
	method := flag.String("method", "blur", "Masking method: blur or blackfill")
	blurStrength := flag.Int("blur-strength", 30, "Gaussian blur strength")
	edgeBlur := flag.Float64("edge-blur", 0.2, "Edge blur size (0.0–1.0)")
	output := flag.String("output", "", "Output file path")

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

	ctx := context.Background()
	data, err := service.MaskImage(ctx, file, opts...)
	if err != nil {
		return fmt.Errorf("masking failed: %w", err)
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

	fmt.Println("Saved to", outPath)
	return nil
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
