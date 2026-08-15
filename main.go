package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	model       = "gemini-3.1-flash-lite-image"
	endpoint    = "https://generativelanguage.googleapis.com/v1beta/interactions"
	maxAttempts = 4
)

var allowedRatios = map[string]bool{
	"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
	"4:5": true, "5:4": true, "9:16": true, "16:9": true, "21:9": true,
}

type inputItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     string `json:"data,omitempty"`
}

type imageFormat struct {
	Type        string `json:"type"`
	MimeType    string `json:"mime_type"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
}

type requestBody struct {
	Model          string      `json:"model"`
	Input          []inputItem `json:"input"`
	ResponseFormat imageFormat `json:"response_format"`
}

type apiImage struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

type apiResponse struct {
	ID          string    `json:"id"`
	OutputImage *apiImage `json:"output_image"`
}

type result struct {
	OK            bool   `json:"ok"`
	Output         string `json:"output"`
	Model          string `json:"model"`
	AspectRatio    string `json:"aspect_ratio,omitempty"`
	InteractionID  string `json:"interaction_id,omitempty"`
	ReferenceImage bool   `json:"reference_image"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "configure" {
		configure()
		return
	}
	prompt := flag.String("prompt", "", "Prompt text (required)")
	promptFile := flag.String("prompt-file", "", "Path to a UTF-8 prompt text file")
	output := flag.String("output", "", "Output PNG path (required)")
	ratio := flag.String("ratio", "", "Optional aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9")
	input := flag.String("input", "", "Optional reference image path")
	timeout := flag.Duration("timeout", 90*time.Second, "Total request timeout")
	flag.Parse()

	if flag.NArg() != 0 || strings.TrimSpace(*output) == "" {
		fail(errors.New("usage: nanobanana [--prompt TEXT | --prompt-file FILE | stdin] --output FILE.png [--ratio RATIO] [--input IMAGE]"))
	}
	if *ratio != "" && !allowedRatios[*ratio] {
		fail(fmt.Errorf("unsupported ratio %q", *ratio))
	}
	if *timeout <= 0 {
		fail(errors.New("timeout must be positive"))
	}
	apiKey, err := apiKey()
	if err != nil { fail(err) }
	promptText, err := readPrompt(*prompt, *promptFile)
	if err != nil { fail(err) }

	items := []inputItem{{Type: "text", Text: promptText}}
	if *input != "" {
		image, err := loadReferenceImage(*input)
		if err != nil { fail(err) }
		items = append(items, image)
	}
	body, err := json.Marshal(requestBody{
		Model: model, Input: items,
		ResponseFormat: imageFormat{Type: "image", MimeType: "image/png", AspectRatio: *ratio},
	})
	if err != nil { fail(fmt.Errorf("encode request: %w", err)) }

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	response, err := callGemini(ctx, apiKey, body)
	if err != nil { fail(err) }
	if response.OutputImage == nil || response.OutputImage.Data == "" {
		fail(errors.New("Gemini returned no generated image"))
	}
	data, err := base64.StdEncoding.DecodeString(response.OutputImage.Data)
	if err != nil { fail(fmt.Errorf("decode generated image: %w", err)) }
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil && filepath.Dir(*output) != "." {
		fail(fmt.Errorf("create output directory: %w", err))
	}
	if err := os.WriteFile(*output, data, 0644); err != nil { fail(fmt.Errorf("write output: %w", err)) }

	printJSON(result{OK: true, Output: *output, Model: model, AspectRatio: *ratio, InteractionID: response.ID, ReferenceImage: *input != ""})
}

func readPrompt(prompt, promptFile string) (string, error) {
	if strings.TrimSpace(prompt) != "" && strings.TrimSpace(promptFile) != "" {
		return "", errors.New("use only one of --prompt or --prompt-file")
	}
	if strings.TrimSpace(prompt) != "" { return prompt, nil }
	if strings.TrimSpace(promptFile) != "" {
		b, err := os.ReadFile(promptFile)
		if err != nil { return "", fmt.Errorf("read prompt file: %w", err) }
		if strings.TrimSpace(string(b)) == "" { return "", errors.New("prompt file is empty") }
		return string(b), nil
	}
	info, err := os.Stdin.Stat()
	if err != nil { return "", fmt.Errorf("inspect standard input: %w", err) }
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("prompt is required; pass --prompt, --prompt-file, or pipe text to standard input")
	}
	b, err := io.ReadAll(io.LimitReader(os.Stdin, 1*1024*1024+1))
	if err != nil { return "", fmt.Errorf("read prompt from standard input: %w", err) }
	if len(b) > 1*1024*1024 { return "", errors.New("prompt from standard input exceeds 1 MB") }
	if strings.TrimSpace(string(b)) == "" { return "", errors.New("prompt from standard input is empty") }
	return string(b), nil
}

type config struct {
	APIKey string `json:"api_key"`
}

// apiKey prefers an environment variable, which is best for CI and agents,
// then falls back to the key saved by `nanobanana configure`.
func apiKey() (string, error) {
	if key := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); key != "" {
		return key, nil
	}
	path, err := configPath()
	if err != nil { return "", err }
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("no Gemini API key found; set GEMINI_API_KEY or run: nanobanana configure")
	}
	if err != nil { return "", fmt.Errorf("read saved configuration: %w", err) }
	var c config
	if err := json.Unmarshal(b, &c); err != nil { return "", fmt.Errorf("read saved configuration: %w", err) }
	if strings.TrimSpace(c.APIKey) == "" { return "", errors.New("saved configuration has no API key; run: nanobanana configure") }
	return strings.TrimSpace(c.APIKey), nil
}

func configure() {
	path, err := configPath()
	if err != nil { fail(err) }
	fmt.Fprint(os.Stderr, "Gemini API key: ")
	var key string
	if _, err := fmt.Fscanln(os.Stdin, &key); err != nil { fail(errors.New("no API key provided")) }
	key = strings.TrimSpace(key)
	if key == "" { fail(errors.New("no API key provided")) }
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { fail(fmt.Errorf("create configuration directory: %w", err)) }
	b, err := json.Marshal(config{APIKey: key})
	if err != nil { fail(fmt.Errorf("encode configuration: %w", err)) }
	if err := os.WriteFile(path, b, 0600); err != nil { fail(fmt.Errorf("save configuration: %w", err)) }
	printJSON(map[string]any{"ok": true, "configured": true, "config_path": path})
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil { return "", fmt.Errorf("locate configuration directory: %w", err) }
	return filepath.Join(dir, "nanobanana", "config.json"), nil
}

func loadReferenceImage(path string) (inputItem, error) {
	data, err := os.ReadFile(path)
	if err != nil { return inputItem{}, fmt.Errorf("read reference image: %w", err) }
	if len(data) == 0 { return inputItem{}, errors.New("reference image is empty") }
	if len(data) > 20*1024*1024 { return inputItem{}, errors.New("reference image exceeds 20 MB") }
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		return inputItem{}, fmt.Errorf("reference file is not a recognized image (%s)", mimeType)
	}
	return inputItem{Type: "image", MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(data)}, nil
}

func callGemini(ctx context.Context, key string, body []byte) (*apiResponse, error) {
	client := &http.Client{}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil { return nil, err }
		req.Header.Set("x-goog-api-key", key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("Gemini request: %w", err)
		} else {
			payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
			resp.Body.Close()
			if readErr != nil { return nil, fmt.Errorf("read Gemini response: %w", readErr) }
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var parsed apiResponse
				if err := json.Unmarshal(payload, &parsed); err != nil { return nil, fmt.Errorf("decode Gemini response: %w", err) }
				return &parsed, nil
			}
			lastErr = fmt.Errorf("Gemini API returned %s: %s", resp.Status, compact(payload))
			if resp.StatusCode != 429 && resp.StatusCode < 500 { return nil, lastErr }
		}
		if attempt < maxAttempts {
			delay := time.Duration(1<<(attempt-1))*time.Second + time.Duration(rand.Intn(250))*time.Millisecond
			select { case <-ctx.Done(): return nil, ctx.Err(); case <-time.After(delay): }
		}
	}
	return nil, lastErr
}

func compact(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 500 { return s[:500] + "…" }
	return s
}

func fail(err error) {
	printJSON(map[string]any{"ok": false, "error": err.Error()})
	os.Exit(1)
}

func printJSON(v any) {
	_ = json.NewEncoder(os.Stdout).Encode(v)
}
