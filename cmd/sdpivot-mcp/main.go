package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName = "SDPivot MCP"
	defaultAPI = "http://localhost:8080"
)

// opVersion is injected by the release build and keeps MCP handshakes on the
// independent OP product version rather than the upstream WeKnora version.
var opVersion = "1.0.0"

type apiClient struct {
	baseURL, token string
	http           *http.Client
}

func main() {
	transport := flag.String("transport", env("SDPIVOT_MCP_TRANSPORT", "stdio"), "stdio, sse, or streamable-http")
	listen := flag.String("listen", env("SDPIVOT_MCP_LISTEN", ":8090"), "HTTP listen address")
	api := flag.String("server", env("SDPIVOT_SERVER", defaultAPI), "SDPivot server URL")
	token := flag.String("token", os.Getenv("SDPIVOT_TOKEN"), "SDPivot access token")
	flag.Parse()
	s := server.NewMCPServer(serverName, opVersion, server.WithToolCapabilities(true))
	c := &apiClient{baseURL: strings.TrimRight(*api, "/"), token: *token, http: &http.Client{Timeout: 90 * time.Second}}
	registerTools(s, c)
	var err error
	switch *transport {
	case "stdio":
		err = server.ServeStdio(s)
	case "sse":
		err = server.NewSSEServer(s, server.WithSSEEndpoint("/sse"), server.WithMessageEndpoint("/message")).Start(*listen)
	case "http", "streamable-http":
		err = server.NewStreamableHTTPServer(s, server.WithEndpointPath("/mcp")).Start(*listen)
	default:
		log.Fatalf("unsupported transport %q", *transport)
	}
	if err != nil {
		log.Fatal(err)
	}
}
func registerTools(s *server.MCPServer, c *apiClient) {
	s.AddTool(mcp.NewTool("search_documents", mcp.WithDescription("Search documents visible to the current SDPivot user."), mcp.WithString("query", mcp.Required())), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return c.toolRequest(ctx, http.MethodGet, "/api/v1/sdp/search?q="+url.QueryEscape(query), nil)
	})
	s.AddTool(mcp.NewTool("list_spaces", mcp.WithDescription("List knowledge spaces visible to the current SDPivot user.")), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return c.toolRequest(ctx, http.MethodGet, "/api/v1/sdp/spaces", nil)
	})
	s.AddTool(mcp.NewTool("get_document", mcp.WithDescription("Get document metadata by ID."), mcp.WithString("document_id", mcp.Required())), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("document_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return c.toolRequest(ctx, http.MethodGet, "/api/v1/sdp/documents/"+url.PathEscape(id), nil)
	})
	s.AddTool(mcp.NewTool("ask_question", mcp.WithDescription("Ask SDPivot a question, optionally scoped to a space."), mcp.WithString("question", mcp.Required()), mcp.WithString("space_id")), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := req.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		created, err := c.request(ctx, http.MethodPost, "/api/v1/sdp/qa/sessions", map[string]string{"title": question, "space_id": req.GetString("space_id", "")})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var response struct {
			Session struct {
				ID string `json:"id"`
			} `json:"session"`
		}
		if err := json.Unmarshal(created, &response); err != nil || response.Session.ID == "" {
			return mcp.NewToolResultError("SDPivot did not return a Q&A session ID"), nil
		}
		return c.toolRequest(ctx, http.MethodPost, "/api/v1/sdp/qa/sessions/"+url.PathEscape(response.Session.ID)+"/messages", map[string]string{"content": question})
	})
}
func (c *apiClient) toolRequest(ctx context.Context, method, path string, body any) (*mcp.CallToolResult, error) {
	data, err := c.request(ctx, method, path, body)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
func (c *apiClient) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SDPivot API returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		return pretty.Bytes(), nil
	}
	return data, nil
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
