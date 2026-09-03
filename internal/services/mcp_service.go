package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"time"

	"github.com/yottaapp/yotta/internal/codexcli"
)

// MCPService integrates the local endpoint with supported desktop clients.
// It never edits client configuration files directly; Codex's own CLI owns
// validation, formatting, and persistence of config.toml.
type MCPService struct{}

func NewMCPService() *MCPService { return &MCPService{} }

func (*MCPService) RegisterCodex(port int) error {
	if port < 1024 || port > 65535 {
		return errors.New("invalid MCP port")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	endpoint := "http://127.0.0.1:" + strconv.Itoa(port) + "/mcp"
	configured, err := readCodexYottaMCP(ctx)
	if err == nil {
		if configured == endpoint {
			return nil
		}
		return fmt.Errorf("codex already has an MCP server named yotta at %s; remove or rename it first", configured)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return err
		}
	}
	cmd, err := codexcli.CommandContext(ctx, "mcp", "add", "yotta", "--url", endpoint)
	if err != nil {
		return errors.New("codex CLI is not installed or is unavailable on PATH")
	}
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return fmt.Errorf("register Yotta with Codex: %w: %s", runErr, boundedCommandOutput(output))
	}
	return nil
}

func (*MCPService) UnregisterCodex(port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	configured, err := readCodexYottaMCP(ctx)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) || errors.Is(err, exec.ErrNotFound) {
			return nil
		}
		return err
	}
	parsed, err := url.Parse(configured)
	if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Path != "/mcp" || parsed.Port() != strconv.Itoa(port) {
		return errors.New("codex yotta entry is not owned by this Yotta endpoint")
	}
	cmd, err := codexcli.CommandContext(ctx, "mcp", "remove", "yotta")
	if err != nil {
		return err
	}
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return fmt.Errorf("remove Yotta from Codex: %w: %s", runErr, boundedCommandOutput(output))
	}
	return nil
}

func readCodexYottaMCP(ctx context.Context) (string, error) {
	cmd, err := codexcli.CommandContext(ctx, "mcp", "get", "yotta", "--json")
	if err != nil {
		return "", exec.ErrNotFound
	}
	raw, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var configured struct {
		Transport struct {
			URL string `json:"url"`
		} `json:"transport"`
	}
	if err := json.Unmarshal(raw, &configured); err != nil || configured.Transport.URL == "" {
		return "", errors.New("codex returned an invalid Yotta MCP configuration")
	}
	return configured.Transport.URL, nil
}

func boundedCommandOutput(raw []byte) string {
	if len(raw) > 4096 {
		raw = raw[len(raw)-4096:]
	}
	return string(raw)
}
