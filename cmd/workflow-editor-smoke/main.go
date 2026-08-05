package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:9227", "WebView2 CDP endpoint")
	screenshot := flag.String("screenshot", ".task/workflow-editor-smoke.png", "PNG output path")
	assetsScreenshot := flag.String("assets-screenshot", ".task/assets-smoke.png", "asset library PNG output path")
	workflowsScreenshot := flag.String("workflows-screenshot", ".task/workflows-smoke.png", "workflow recovery PNG output path")
	launcherScreenshot := flag.String("launcher-screenshot", ".task/launcher-smoke.png", "floating launcher PNG output path")
	schedulesScreenshot := flag.String("schedules-screenshot", ".task/schedules-smoke.png", "schedule editor PNG output path")
	subgraphScreenshot := flag.String("subgraph-screenshot", ".task/subgraph-smoke.png", "subgraph authoring PNG output path")
	captureOnly := flag.Bool("capture-only", false, "capture the current WebView page without running the product journey")
	scheduleEditorOnly := flag.Bool("schedule-editor-only", false, "open and capture the schedule editor without running the full product journey")
	urlContains := flag.String("url-contains", "wails.localhost", "substring used to select one WebView page target")
	seedRoot := flag.String("seed-root", "", "seed the isolated storage root used by the product journey")
	retentionSource := flag.String("retention-source", "", "read-only golden Workflow Source to validate and seed")
	retentionBlobs := flag.String("retention-blobs", "", "read-only Blob Store used by the golden Workflow Source")
	retentionWorkflowID := flag.String("retention-workflow-id", "", "open one seeded golden Workflow instead of the synthetic authoring journey")
	firstScreenBudget := flag.Duration("first-screen-budget", 5*time.Second, "maximum time from CDP connection to a hydrated first screen")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if *seedRoot != "" {
		if err := seedRecoveryFixture(ctx, *seedRoot, *retentionSource, *retentionBlobs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *retentionSource != "" {
		report, err := checkWorkflowRetention(ctx, *retentionSource, *retentionBlobs)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(encoded))
		return
	}
	if *captureOnly {
		if err := captureCurrent(ctx, *endpoint, *urlContains, *screenshot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *scheduleEditorOnly {
		if err := captureScheduleEditor(ctx, *endpoint, *schedulesScreenshot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(ctx, *endpoint, *screenshot, *assetsScreenshot, *workflowsScreenshot, *launcherScreenshot, *schedulesScreenshot, *subgraphScreenshot, *retentionWorkflowID, *firstScreenBudget); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
