package mcpserver

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestRuntimeHotStartsLoopbackStreamableHTTP(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()

	runtime, err := NewRuntime(testApplication(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Start(RuntimeConfig{Enabled: true, Port: port}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runtime.Close(ctx)
	})

	protocolClient, err := client.NewStreamableHttpClient("http://127.0.0.1:" + strconv.Itoa(port) + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = protocolClient.Close() })
	if err := protocolClient.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{Name: "yotta-runtime-test", Version: "1"}
	if _, err := protocolClient.Initialize(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if tools, err := protocolClient.ListTools(context.Background(), mcp.ListToolsRequest{}); err != nil || len(tools.Tools) != 9 {
		t.Fatalf("tools = %#v, err = %v", tools, err)
	}
}

func TestRuntimePrepareRejectsOccupiedPortWithoutChangingSettings(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	runtime, err := NewRuntime(testApplication(t))
	if err != nil {
		t.Fatal(err)
	}
	port := occupied.Addr().(*net.TCPAddr).Port
	if _, _, err := runtime.Prepare(RuntimeConfig{Enabled: true, Port: port}); err == nil {
		t.Fatal("occupied MCP port was accepted")
	}
}
