package nlp

// Engine is the NLP orchestrator. It tries the rule-based parser first,
// then optionally falls back to Ollama.
type Engine struct {
	Ollama *OllamaClient // nil if Ollama is disabled
}

// NewEngine creates an NLP engine. Pass nil for ollama to disable LLM fallback.
func NewEngine(ollama *OllamaClient) *Engine {
	return &Engine{Ollama: ollama}
}

// Process tries to parse the input. Returns a command synchronously if Tier 1
// succeeds. If Tier 1 fails and Ollama is available, it fires an async query
// and returns (nil, nil) — the caller should poll Ollama.ResultCh.
// If both tiers are unavailable, returns (nil, error).
func (e *Engine) Process(input string, callsigns, waypoints []string) (*ParsedCommand, error) {
	// Tier 1: rule-based parser
	cmd, err := Parse(input, callsigns)
	if err == nil {
		return cmd, nil
	}

	// Tier 2: Ollama fallback (async)
	if e.Ollama != nil {
		e.Ollama.QueryAsync(input, callsigns, waypoints)
		return nil, nil // caller polls ResultCh
	}

	return nil, err // No LLM available — return parser error
}
