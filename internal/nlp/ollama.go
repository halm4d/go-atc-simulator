package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrModelNotFound is returned by Ping when the server is reachable but the model is not pulled.
var ErrModelNotFound = errors.New("model not found")

// OllamaResult is the outcome of an async LLM query.
type OllamaResult struct {
	Command *ParsedCommand
	Err     error
}

// OllamaClient handles async communication with a local Ollama instance.
type OllamaClient struct {
	Endpoint string
	Model    string
	ResultCh chan OllamaResult
}

// NewOllamaClient creates a client. ResultCh is buffered(1) so the goroutine never blocks.
func NewOllamaClient(endpoint, model string) *OllamaClient {
	return &OllamaClient{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Model:    model,
		ResultCh: make(chan OllamaResult, 1),
	}
}

// Ping tests connectivity and verifies the configured model is available.
func (o *OllamaClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]string{"name": o.Model})
	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/show", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrModelNotFound, o.Model)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}
	return nil
}

// PullModel downloads the configured model from Ollama. Blocks until complete.
func (o *OllamaClient) PullModel() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	body, _ := json.Marshal(map[string]interface{}{
		"name":   o.Model,
		"stream": false,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	// Drain the response body (Ollama returns status JSON)
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama pull returned status %d", resp.StatusCode)
	}
	return nil
}

// QueryAsync sends the input to Ollama in a goroutine. Poll ResultCh for the answer.
func (o *OllamaClient) QueryAsync(input string, callsigns, waypoints []string) {
	go func() {
		cmd, err := o.query(input, callsigns, waypoints)
		o.ResultCh <- OllamaResult{Command: cmd, Err: err}
	}()
}

func (o *OllamaClient) query(input string, callsigns, waypoints []string) (*ParsedCommand, error) {
	systemPrompt := fmt.Sprintf(`You are an ATC command parser. Extract the structured command from this ATC instruction.
Active aircraft callsigns: %s
Active waypoints: %s

Respond ONLY with JSON in this format:
{"callsign": "XXX000", "command": "heading|altitude|speed|direct|takeoff|lineup|land|hold", "value": <number or string or null>}
If the input is not a valid ATC command, respond with: {"error": "unrecognized"}`,
		strings.Join(callsigns, ", "),
		strings.Join(waypoints, ", "))

	body := map[string]interface{}{
		"model":  o.Model,
		"system": systemPrompt,
		"prompt": input,
		"stream": false,
		"format": "json",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Ollama wraps the model output in {"response": "..."}
	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return parseOllamaJSON(ollamaResp.Response)
}

// parseOllamaJSON parses the model's JSON output into a ParsedCommand.
func parseOllamaJSON(raw string) (*ParsedCommand, error) {
	var result struct {
		Callsign string      `json:"callsign"`
		Command  string      `json:"command"`
		Value    interface{} `json:"value"`
		Error    string      `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("LLM: %s", result.Error)
	}
	if result.Callsign == "" || result.Command == "" {
		return nil, fmt.Errorf("incomplete response from LLM")
	}

	cmd := &ParsedCommand{
		Callsign:    strings.ToUpper(result.Callsign),
		CommandType: strings.ToLower(result.Command),
	}

	// Map value based on command type
	switch cmd.CommandType {
	case "heading", "altitude", "speed":
		switch v := result.Value.(type) {
		case float64:
			cmd.NumValue = v
		case string:
			// Try to parse string as number
			fmt.Sscanf(v, "%f", &cmd.NumValue)
		}
	case "direct":
		switch v := result.Value.(type) {
		case string:
			cmd.StrValue = strings.ToUpper(v)
		}
	}

	return cmd, nil
}
