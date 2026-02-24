package qr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wamuir/svg-qr-code"
)

const webPanelURL = "https://nubecita.casa"

// Run generates an SVG file containing a QR code for the web panel URL
// and writes it to outPath. The file is created with mode 0644.
func Run(outPath string) error {
	qr, err := qrsvg.New(webPanelURL)
	if err != nil {
		return fmt.Errorf("generate qr: %w", err)
	}

	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	if err := os.WriteFile(outPath, []byte(qr.String()), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	return nil
}
