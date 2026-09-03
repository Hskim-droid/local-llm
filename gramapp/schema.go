package main

func chunkJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"heading_hint": map[string]any{"type": "string"},
			"facts":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required":             []string{"facts"},
		"additionalProperties": false,
	}
}

func assembleJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":     map[string]any{"type": "string"},
			"date":      map[string]any{"type": "string"},
			"time":      map[string]any{"type": "string"},
			"place":     map[string]any{"type": "string"},
			"attendees": map[string]any{"type": "string"},
			"agenda":    map[string]any{"type": "string"},
			"exec":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"sections": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"heading": map[string]any{"type": "string"},
						"facts":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required":             []string{"heading", "facts"},
					"additionalProperties": false,
				},
			},
			"actions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required":             []string{"title", "sections"},
		"additionalProperties": false,
	}
}
