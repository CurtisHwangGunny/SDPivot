package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	serverName    = "SDPivot·文枢 MCP"
	serverVersion = "2.0.0"
	defaultAPI    = "http://localhost:8080"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpServer struct {
	api   string
	token string
	http  *http.Client
}

func main() {
	transport := flag.String("transport", env("SDPIVOT_MCP_TRANSPORT", "stdio"), "stdio or http")
	listen := flag.String("listen", env("SDPIVOT_MCP_LISTEN", ":8090"), "HTTP listen address")
	api := flag.String("server", env("SDPIVOT_SERVER", defaultAPI), "SDPivot server URL")
	token := flag.String("token", os.Getenv("SDPIVOT_TOKEN"), "SDPivot access token")
	flag.Parse()

	s := &mcpServer{api: strings.TrimRight(*api, "/"), token: *token, http: &http.Client{Timeout: 90 * time.Second}}
	switch *transport {
	case "stdio":
		if err := s.serveStdio(os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
	case "http":
		mux := http.NewServeMux()
		mux.HandleFunc("/mcp", s.handleHTTP)
		log.Printf("%s listening on %s/mcp", serverName, *listen)
		log.Fatal(http.ListenAndServe(*listen, mux))
	default:
		log.Fatalf("unsupported transport %q", *transport)
	}
}

func (s *mcpServer) serveStdio(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := encoder.Encode(errorResponse(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		if err := encoder.Encode(s.handle(req)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *mcpServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req rpcRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeJSON(w, errorResponse(nil, -32700, "parse error"))
		return
	}
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, s.handle(req))
}

func (s *mcpServer) handle(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return successResponse(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": serverName, "version": serverVersion},
		})
	case "ping":
		return successResponse(req.ID, map[string]any{})
	case "tools/list":
		return successResponse(req.ID, map[string]any{"tools": toolDefinitions()})
	case "tools/call":
		result, err := s.callTool(req.Params)
		if err != nil {
			return successResponse(req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true})
		}
		return successResponse(req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": string(result)}}})
	default:
		return errorResponse(req.ID, -32601, "method not found")
	}
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{"name": "search_documents", "description": "Search documents visible to the current SDPivot user.", "inputSchema": objectSchema(map[string]any{"query": stringSchema("Search query")}, []string{"query"})},
		{"name": "ask_question", "description": "Ask SDPivot·文枢 a question, optionally scoped to a space.", "inputSchema": objectSchema(map[string]any{"question": stringSchema("Question"), "space_id": stringSchema("Optional space ID")}, []string{"question"})},
		{"name": "list_spaces", "description": "List knowledge spaces visible to the current user.", "inputSchema": objectSchema(map[string]any{}, nil)},
		{"name": "get_document", "description": "Get document metadata by ID.", "inputSchema": objectSchema(map[string]any{"document_id": stringSchema("Document ID")}, []string{"document_id"})},
	}
}

func (s *mcpServer) callTool(raw json.RawMessage) ([]byte, error) {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &call); err != nil {
		return nil, errors.New("invalid tool arguments")
	}
	switch call.Name {
	case "search_documents":
		query, err := requiredString(call.Arguments, "query")
		if err != nil {
			return nil, err
		}
		return s.request(http.MethodGet, "/api/v1/sdp/search?q="+url.QueryEscape(query), nil)
	case "list_spaces":
		return s.request(http.MethodGet, "/api/v1/sdp/spaces", nil)
	case "get_document":
		documentID, err := requiredString(call.Arguments, "document_id")
		if err != nil {
			return nil, err
		}
		return s.request(http.MethodGet, "/api/v1/sdp/documents/"+url.PathEscape(documentID), nil)
	case "ask_question":
		question, err := requiredString(call.Arguments, "question")
		if err != nil {
			return nil, err
		}
		spaceID, _ := call.Arguments["space_id"].(string)
		return s.askQuestion(question, spaceID)
	default:
		return nil, fmt.Errorf("unknown tool %q", call.Name)
	}
}

func (s *mcpServer) askQuestion(question, spaceID string) ([]byte, error) {
	created, err := s.request(http.MethodPost, "/api/v1/sdp/qa/sessions", map[string]string{"title": question, "space_id": spaceID})
	if err != nil {
		return nil, err
	}
	var response struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.Unmarshal(created, &response); err != nil || response.Session.ID == "" {
		return nil, errors.New("SDPivot did not return a Q&A session ID")
	}
	path := "/api/v1/sdp/qa/sessions/" + url.PathEscape(response.Session.ID) + "/messages"
	return s.request(http.MethodPost, path, map[string]string{"content": question})
}

func (s *mcpServer) request(method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, s.api+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.http.Do(req)
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

func successResponse(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func requiredString(values map[string]any, key string) (string, error) {
	value, _ := values[key].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]string {
	return map[string]string{"type": "string", "description": description}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
