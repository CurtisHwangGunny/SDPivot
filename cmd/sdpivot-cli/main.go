package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultServer = "http://localhost:8080"

type config struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

type client struct {
	server string
	token  string
	http   *http.Client
}

func main() {
	server, args, err := parseServer(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	cfg, err := loadConfig()
	if err != nil {
		fatal(err)
	}
	if server == "" {
		server = os.Getenv("SDPIVOT_SERVER")
	}
	if server == "" {
		server = cfg.Server
	}
	if server == "" {
		server = defaultServer
	}
	c := &client{server: strings.TrimRight(server, "/"), token: cfg.Token, http: &http.Client{Timeout: 60 * time.Second}}

	switch args[0] {
	case "auth":
		err = runAuth(c, args[1:])
	case "search":
		err = runSearch(c, args[1:])
	case "doc":
		err = runDoc(c, args[1:])
	case "qa":
		err = runQA(c, args[1:])
	case "help", "--help", "-h":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		fatal(err)
	}
}

func parseServer(args []string) (string, []string, error) {
	result := make([]string, 0, len(args))
	var server string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--server=") {
			server = strings.TrimPrefix(arg, "--server=")
			continue
		}
		if arg == "--server" {
			if i+1 >= len(args) {
				return "", nil, errors.New("--server requires a URL")
			}
			server = args[i+1]
			i++
			continue
		}
		result = append(result, arg)
	}
	return server, result, nil
}

func runAuth(c *client, args []string) error {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	email := fs.String("email", "", "login email")
	password := fs.String("password", "", "login password")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *email == "" || *password == "" {
		return errors.New("auth requires --email and --password")
	}
	var response struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := c.jsonRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": *email, "password": *password}, &response); err != nil {
		return err
	}
	token := response.AccessToken
	if token == "" {
		token = response.Token
	}
	if token == "" {
		return errors.New("login response did not contain an access token")
	}
	if err := saveConfig(config{Server: c.server, Token: token}); err != nil {
		return err
	}
	fmt.Println("Authenticated with SDPivot·文枢; token saved to the CLI configuration.")
	return nil
}

func runSearch(c *client, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	query := fs.String("query", "", "search query")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*query) == "" {
		return errors.New("search requires --query")
	}
	return c.jsonRequest(http.MethodGet, "/api/v1/sdp/search?q="+url.QueryEscape(*query), nil, os.Stdout)
}

func runDoc(c *client, args []string) error {
	if len(args) == 0 {
		return errors.New("doc requires list or upload")
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("doc list", flag.ContinueOnError)
		space := fs.String("space", "", "space ID")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *space == "" {
			return errors.New("doc list requires --space")
		}
		return c.jsonRequest(http.MethodGet, "/api/v1/sdp/documents?space_id="+url.QueryEscape(*space), nil, os.Stdout)
	case "upload":
		fs := flag.NewFlagSet("doc upload", flag.ContinueOnError)
		space := fs.String("space", "", "space ID")
		path := fs.String("file", "", "file path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *space == "" || *path == "" {
			return errors.New("doc upload requires --space and --file")
		}
		return c.upload(*space, *path)
	default:
		return fmt.Errorf("unknown doc command %q", args[0])
	}
}

func runQA(c *client, args []string) error {
	fs := flag.NewFlagSet("qa", flag.ContinueOnError)
	session := fs.String("session", "", "Q&A session ID")
	content := fs.String("content", "", "question content")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *session == "" || strings.TrimSpace(*content) == "" {
		return errors.New("qa requires --session and --content")
	}
	path := "/api/v1/sdp/qa/sessions/" + url.PathEscape(*session) + "/messages"
	return c.jsonRequest(http.MethodPost, path, map[string]string{"content": *content}, os.Stdout)
}

func (c *client) jsonRequest(method, path string, body any, output any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.server+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, output)
}

func (c *client) upload(space, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("space_id", space); err != nil {
		return err
	}
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.server+"/api/v1/sdp/documents/upload", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, os.Stdout)
}

func decodeResponse(resp *http.Response, output any) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SDPivot API returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if writer, ok := output.(io.Writer); ok {
		var pretty bytes.Buffer
		if json.Indent(&pretty, data, "", "  ") == nil {
			data = pretty.Bytes()
		}
		_, err = fmt.Fprintln(writer, string(data))
		return err
	}
	return json.Unmarshal(data, output)
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sdpivot", "config.json"), nil
}

func loadConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return config{}, nil
	}
	if err != nil {
		return config{}, err
	}
	var cfg config
	return cfg, json.Unmarshal(data, &cfg)
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func usage() {
	fmt.Fprintln(os.Stderr, `SDPivot·文枢 CLI

Usage:
  sdpivot-cli [--server URL] auth --email EMAIL --password PASSWORD
  sdpivot-cli [--server URL] search --query QUERY
  sdpivot-cli [--server URL] doc list --space SPACE_ID
  sdpivot-cli [--server URL] doc upload --space SPACE_ID --file PATH
  sdpivot-cli [--server URL] qa --session SESSION_ID --content CONTENT`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
