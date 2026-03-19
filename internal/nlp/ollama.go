package nlp

import (
	"atc-sim/internal/logger"
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

// validCommands is the canonical list of command types the game accepts.
// The index+1 maps to the numbered choice used in the LLM prompt.
var validCommands = [8]string{"heading", "altitude", "speed", "direct", "takeoff", "lineup", "land", "hold"}

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
	logger.Debug("pinging ollama", "endpoint", o.Endpoint, "model", o.Model)
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
		logger.Warn("ollama unreachable", "error", err)
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		logger.Info("model not found on ollama", "model", o.Model)
		return fmt.Errorf("%w: %s", ErrModelNotFound, o.Model)
	}
	if resp.StatusCode != http.StatusOK {
		logger.Warn("ollama ping unexpected status", "status", resp.StatusCode)
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}
	logger.Info("ollama ping OK", "model", o.Model)
	return nil
}

// PullModel downloads the configured model from Ollama. Blocks until complete.
func (o *OllamaClient) PullModel() error {
	logger.Info("pulling model from ollama", "model", o.Model)
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
		logger.Error("failed to pull model", "model", o.Model, "error", err)
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	// Drain the response body (Ollama returns status JSON)
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Error("ollama pull failed", "model", o.Model, "status", resp.StatusCode)
		return fmt.Errorf("ollama pull returned status %d", resp.StatusCode)
	}
	logger.Info("model pulled successfully", "model", o.Model)
	return nil
}

// QueryAsync sends the input to Ollama in a goroutine. Poll ResultCh for the answer.
func (o *OllamaClient) QueryAsync(input string, callsigns, waypoints []string) {
	go func() {
		cmd, err := o.query(input, callsigns, waypoints)
		o.ResultCh <- OllamaResult{Command: cmd, Err: err}
	}()
}

func (o *OllamaClient) query(input string, callsigns, waypoints []string) (cmd *ParsedCommand, err error) {
	logger.Debug("ollama query", "input", input)
	defer func() {
		if err != nil {
			logger.Warn("ollama query failed", "input", input, "error", err)
		} else {
			logger.Debug("ollama query result", "callsign", cmd.Callsign, "command", cmd.CommandType)
		}
	}()
	systemPrompt := fmt.Sprintf(`Parse ATC instruction into JSON.

Commands: 1=heading 2=altitude 3=speed 4=direct 5=takeoff 6=lineup 7=land 8=hold
Callsigns: %s
Waypoints: %s

Examples:
"DAL986 turn heading 270" -> {"callsign":"DAL986","command":1,"value":270}
"BAW123 go to NORAH" -> {"callsign":"BAW123","command":4,"value":"NORAH"}
"AAL789 climb 5000" -> {"callsign":"AAL789","command":2,"value":5000}
"SWA100 speed 250" -> {"callsign":"SWA100","command":3,"value":250}
"DLH456 cleared takeoff" -> {"callsign":"DLH456","command":5,"value":null}
"UAE22 hold" -> {"callsign":"UAE22","command":8,"value":null}

Reply with ONLY JSON:`,
		strings.Join(callsigns, ", "),
		strings.Join(waypoints, ", "))

	body := map[string]interface{}{
		"model":  o.Model,
		"system": systemPrompt,
		"prompt": input,
		"stream": false,
		"format": "json",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ollamaResp, err := o.callOllama(ctx, body)
	if err != nil {
		return nil, err
	}

	logger.Debug("ollama raw response", "response", ollamaResp)
	cmd, rawCommand, err := parseOllamaJSON(ollamaResp, callsigns)
	if err != nil {
		return nil, err
	}

	// If command resolved successfully, return immediately
	if cmd.CommandType != "" {
		return cmd, nil
	}

	// Command didn't resolve — retry with a focused classification prompt
	cmdType, ok := o.classifyRetry(ctx, input, rawCommand)
	if !ok {
		return nil, fmt.Errorf("unrecognized command from LLM: %v", rawCommand)
	}
	cmd.CommandType = cmdType

	// Re-map value now that we know the command type
	var result ollamaJSON
	json.Unmarshal([]byte(ollamaResp), &result)
	switch cmd.CommandType {
	case "heading", "altitude", "speed":
		switch v := result.Value.(type) {
		case float64:
			cmd.NumValue = v
		case string:
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

// callOllama sends a request to the Ollama API and returns the model's response string.
func (o *OllamaClient) callOllama(ctx context.Context, body map[string]interface{}) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}

// ollamaJSON is the raw JSON structure returned by the LLM.
type ollamaJSON struct {
	Callsign string      `json:"callsign"`
	Command  interface{} `json:"command"`
	Value    interface{} `json:"value"`
	Error    string      `json:"error"`
}

// parseOllamaJSON parses the model's JSON output into a ParsedCommand.
// Returns the parsed command and the raw command value (for retry if resolution fails).
func parseOllamaJSON(raw string, callsigns []string) (*ParsedCommand, interface{}, error) {
	var result ollamaJSON
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}
	if result.Error != "" {
		return nil, nil, fmt.Errorf("LLM: %s", result.Error)
	}
	if result.Callsign == "" || result.Command == nil {
		return nil, nil, fmt.Errorf("incomplete response from LLM")
	}

	cmdType, ok := resolveCommandType(result.Command)

	cmd := &ParsedCommand{
		Callsign:    fuzzyMatchCallsign(result.Callsign, callsigns),
		CommandType: cmdType,
	}

	if ok {
		// Map value based on command type
		switch cmd.CommandType {
		case "heading", "altitude", "speed":
			switch v := result.Value.(type) {
			case float64:
				cmd.NumValue = v
			case string:
				fmt.Sscanf(v, "%f", &cmd.NumValue)
			}
		case "direct":
			switch v := result.Value.(type) {
			case string:
				cmd.StrValue = strings.ToUpper(v)
			}
		}
	}

	return cmd, result.Command, nil
}

// resolveCommandType resolves the LLM's command field (number or string) to a valid command name.
// Returns the command name and true if resolved, or empty string and false if not.
func resolveCommandType(raw interface{}) (string, bool) {
	switch v := raw.(type) {
	case float64:
		idx := int(v) - 1
		if idx >= 0 && idx < len(validCommands) {
			return validCommands[idx], true
		}
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		// Check if it's already a valid command name
		for _, c := range validCommands {
			if lower == c {
				return c, true
			}
		}
		// Try parsing as a number string (e.g. "4")
		var n int
		if _, err := fmt.Sscanf(lower, "%d", &n); err == nil {
			idx := n - 1
			if idx >= 0 && idx < len(validCommands) {
				return validCommands[idx], true
			}
		}
	}
	return "", false
}

// classifyRetry makes a focused second LLM call to classify the command type
// when the first response had an unrecognized command.
func (o *OllamaClient) classifyRetry(ctx context.Context, input string, rawCommand interface{}) (string, bool) {
	logger.Debug("ollama classify retry", "input", input, "rawCommand", rawCommand)

	prompt := fmt.Sprintf(`The ATC command was: "%s"
Pick the correct command number:
1=heading 2=altitude 3=speed 4=direct 5=takeoff 6=lineup 7=land 8=hold
Reply ONLY with JSON: {"command": <number>}`, input)

	body := map[string]interface{}{
		"model":  o.Model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}

	raw, err := o.callOllama(ctx, body)
	if err != nil {
		return "", false
	}

	logger.Debug("ollama classify retry response", "response", raw)

	var retryResult struct {
		Command interface{} `json:"command"`
	}
	if err := json.Unmarshal([]byte(raw), &retryResult); err != nil {
		return "", false
	}

	return resolveCommandType(retryResult.Command)
}

// fuzzyMatchCallsign finds the closest matching callsign from the active list.
// Returns the original (uppercased) if no close match is found.
func fuzzyMatchCallsign(raw string, callsigns []string) string {
	raw = strings.ToUpper(raw)
	if len(callsigns) == 0 {
		return raw
	}
	for _, cs := range callsigns {
		if strings.ToUpper(cs) == raw {
			return strings.ToUpper(cs)
		}
	}
	best := raw
	bestDist := 3 // threshold: only accept distance <= 2
	for _, cs := range callsigns {
		d := editDistance(raw, strings.ToUpper(cs))
		if d < bestDist {
			bestDist = d
			best = strings.ToUpper(cs)
		}
	}
	return best
}

// editDistance computes the Levenshtein distance between two strings.
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}
