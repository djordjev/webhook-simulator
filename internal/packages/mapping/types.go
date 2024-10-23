package mapping

type Mapper interface {
	Refresh() error
	GetMappings() []Flow
}

type RequestDefinition struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Body    map[string]any    `json:"body"`
	Headers map[string]string `json:"headers"`
}

type ResponseDefinition struct {
	Code           int               `json:"code"`
	Delay          int               `json:"delay"`
	IncludeRequest bool              `json:"includeRequest"`
	Headers        map[string]string `json:"headers"`
	Body           map[string]any    `json:"body"`
}

type WebHookDefinition struct {
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Delay          int               `json:"delay"`
	IncludeRequest bool              `json:"includeRequest"`
	Headers        map[string]string `json:"headers"`
	Body           map[string]any    `json:"body"`
}

type Flow struct {
	Request  *RequestDefinition  `json:"request"`
	Response *ResponseDefinition `json:"response"`
	WebHook  *WebHookDefinition  `json:"web_hook"`
}
