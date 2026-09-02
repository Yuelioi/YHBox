//go:build windows

package noderuntime

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/pkg/capture"
	"github.com/yottaapp/yotta/pkg/vision"
	"github.com/yottaapp/yotta/pkg/winutil"
	_ "modernc.org/sqlite"
)

// TestNativeTemplateDiagnostic is an opt-in real-target diagnostic. The wrapper
// in scripts/template-match-diagnostic.ps1 supplies the exact local root,
// workflow, resource and target selectors.
func TestNativeTemplateDiagnostic(t *testing.T) {
	if os.Getenv("YOTTA_TEMPLATE_DIAGNOSTIC") != "1" {
		t.Skip("set YOTTA_TEMPLATE_DIAGNOSTIC=1")
	}
	root := mustEnv(t, "YOTTA_DIAGNOSTIC_ROOT")
	workflowName := mustEnv(t, "YOTTA_DIAGNOSTIC_WORKFLOW")
	resourceID := mustEnv(t, "YOTTA_DIAGNOSTIC_RESOURCE")
	variantID := envOr("YOTTA_DIAGNOSTIC_VARIANT", "default")
	outputDir := mustEnv(t, "YOTTA_DIAGNOSTIC_OUTPUT")

	source := readWorkflowSource(t, root, workflowName)
	variant := findImageVariant(t, source, resourceID, variantID)
	templateBytes := readObject(t, root, variant.Blob.Digest.String(), variant.Blob.Size)

	executable := mustEnv(t, "YOTTA_DIAGNOSTIC_EXECUTABLE")
	spec := winutil.MatchSpec{
		Title:      mustEnv(t, "YOTTA_DIAGNOSTIC_WINDOW_TITLE"),
		TitleMatch: envOr("YOTTA_DIAGNOSTIC_TITLE_MATCH", "exact"),
		Class:      mustEnv(t, "YOTTA_DIAGNOSTIC_WINDOW_CLASS"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	window, err := winutil.ResolveExecutableWindow(ctx, executable, spec, "unique", 3*time.Second, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	backendName := envOr("YOTTA_DIAGNOSTIC_CAPTURE_BACKEND", "gdi")
	backend, warning, err := capture.NewIBackend(backendName)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	frame, err := backend.Frame(capture.Handle(window.HWND))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join(outputDir, "capture.png"), frame)
	writeBytes(t, filepath.Join(outputDir, "template-original.png"), templateBytes)

	targetResolution := [2]int{frame.Bounds().Dx(), frame.Bounds().Dy()}
	scaledBytes := scaleTemplate(t, templateBytes, variant.Resolution, targetResolution)
	writeBytes(t, filepath.Join(outputDir, "template-scaled.png"), scaledBytes)
	original := runDiagnosticMatch(t, frame, templateBytes)
	scaled := runDiagnosticMatch(t, frame, scaledBytes)

	report := map[string]any{
		"workflow": workflowName, "resourceId": resourceID, "variantId": variantID,
		"sourceResolution": variant.Resolution, "bbox": variant.BBox,
		"frameResolution": targetResolution, "captureBackend": backend.Name(), "captureWarning": warning,
		"templateBlobDigest": variant.Blob.Digest.String(), "templateBlobBytes": len(templateBytes),
		"scaledTemplateBytes": len(scaledBytes), "threshold": 0.85,
		"original": original, "scaled": scaled,
	}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	writeBytes(t, filepath.Join(outputDir, "report.json"), encoded)
	t.Logf("\n%s", encoded)
	if !scaled.Matched {
		t.Fatalf("2K scaled template did not match: score=%.6f threshold=0.85 evidence=%s", scaled.Score, outputDir)
	}
}

func readWorkflowSource(t *testing.T, root, name string) schema.WorkflowSource {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(root, "catalog", "content.db")) + "?mode=ro&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var raw []byte
	if err := db.QueryRow(`SELECT artifact FROM workflow_sources WHERE name = ?`, name).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var source schema.WorkflowSource
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	return source
}

func findImageVariant(t *testing.T, source schema.WorkflowSource, resourceID, variantID string) schema.ImageResourceVariant {
	t.Helper()
	for _, resource := range source.Resources {
		if resource.ID != resourceID || resource.Image == nil {
			continue
		}
		for _, variant := range resource.Image.Variants {
			if variant.ID == variantID {
				return variant
			}
		}
	}
	t.Fatalf("image resource %q variant %q not found", resourceID, variantID)
	return schema.ImageResourceVariant{}
}

func readObject(t *testing.T, root, digest string, expected int64) []byte {
	t.Helper()
	name := strings.TrimPrefix(digest, "sha256:")
	if len(name) != 64 {
		t.Fatalf("invalid digest %q", digest)
	}
	content, err := os.ReadFile(filepath.Join(root, "objects", "sha256", name[:2], name[2:4], name))
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(content)) != expected {
		t.Fatalf("blob size=%d want=%d", len(content), expected)
	}
	return content
}

func scaleTemplate(t *testing.T, raw []byte, sourceResolution, targetResolution [2]int) []byte {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	rgba := image.NewRGBA(decoded.Bounds())
	draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	gray, width, height := vision.RGBAToGray(rgba)
	scale := float64(targetResolution[0]) / float64(sourceResolution[0])
	targetW, targetH := int(math.Round(float64(width)*scale)), int(math.Round(float64(height)*scale))
	resized := vision.ResizeGray(gray, width, height, targetW, targetH)
	out := image.NewGray(image.Rect(0, 0, targetW, targetH))
	for i, value := range resized {
		out.Pix[i] = uint8(math.Round(float64(value) * 255))
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, out); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func runDiagnosticMatch(t *testing.T, frame *image.RGBA, raw []byte) visionMatchResult {
	t.Helper()
	result, err := matchTemplateFrame(frame, raw, visionRegion{X: 0, Y: 0, Width: 1, Height: 1, Unit: "ratio"}, 0.85)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writePNG(t *testing.T, path string, value image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, value); err != nil {
		t.Fatal(err)
	}
}
func writeBytes(t *testing.T, path string, value []byte) {
	t.Helper()
	if err := os.WriteFile(path, value, 0o600); err != nil {
		t.Fatal(err)
	}
}
func mustEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s is required", key)
	}
	return value
}
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
