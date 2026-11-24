package coach

import "encoding/json"

// toolDefs returns the JSON-schema definitions of every coach tool. Kept
// few and simple on purpose: providers vary in tool-calling quality, and
// strict schemas with one retry cover malformed calls.
func toolDefs() []ToolDef {
	return []ToolDef{
		{
			Type: "function",
			Name: "create_goal",
			Description: "Create a structured training goal for the athlete. " +
				"Run goals use targetMetric \"distance_km\" and a targetValue in km (e.g. \"10\" or \"21.1\"). " +
				"Boulder goals use targetMetric \"grade\" and a Font boulder grade (e.g. \"6B\", \"7A\", \"7C\"). " +
				"Lead goals use targetMetric \"grade\" and a Font lead grade (e.g. \"6a\", \"7a+\").",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"activityType": map[string]any{"type": "string", "enum": []string{"run", "boulder", "lead"}},
					"targetMetric": map[string]any{"type": "string", "enum": []string{"distance_km", "grade"}},
					"targetValue":  map[string]any{"type": "string", "description": "km value or Font grade"},
					"targetDate":   map[string]any{"type": "string", "description": "YYYY-MM-DD deadline, optional"},
					"notes":        map[string]any{"type": "string"},
					"isPrimary":    map[string]any{"type": "boolean", "description": "the goal plans are built around"},
				},
				"required": []string{"activityType", "targetMetric", "targetValue"},
			}),
		},
		{
			Type: "function",
			Name: "set_primary_goal",
			Description: "Make an existing goal the primary one (plans are built around the primary goal). " +
				"Clears the primary flag of other active goals.",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"goalId": map[string]any{"type": "string"},
				},
				"required": []string{"goalId"},
			}),
		},
		{
			Type:        "function",
			Name:        "update_goal_status",
			Description: "Update a goal's status (active, achieved, abandoned).",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"goalId": map[string]any{"type": "string"},
					"status": map[string]any{"type": "string", "enum": []string{"active", "achieved", "abandoned"}},
				},
				"required": []string{"goalId", "status"},
			}),
		},
		{
			Type: "function",
			Name: "create_training_plan",
			Description: "Create a new version of the athlete's training plan (archives the current active plan). " +
				"Build the plan around the primary goal; secondary goals act as constraints. " +
				"Weeks have consecutive weekNumbers starting at 1; sessions carry a sport (run, boulder, lead, strength, other), " +
				"session type, planned date (YYYY-MM-DD) and optional targets.",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"goalId": map[string]any{"type": "string", "description": "primary goal this plan targets"},
					"title":  map[string]any{"type": "string"},
					"notes":  map[string]any{"type": "string"},
					"weeks": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"weekNumber": map[string]any{"type": "integer"},
								"startDate":  map[string]any{"type": "string", "description": "YYYY-MM-DD (Monday)"},
								"focus":      map[string]any{"type": "string"},
								"sessions": map[string]any{
									"type": "array",
									"items": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"sport":             map[string]any{"type": "string", "enum": []string{"run", "boulder", "lead", "strength", "other"}},
											"sessionType":       map[string]any{"type": "string"},
											"description":       map[string]any{"type": "string"},
											"plannedDate":       map[string]any{"type": "string", "description": "YYYY-MM-DD"},
											"targetDurationMin": map[string]any{"type": "integer"},
											"targetDistanceKm":  map[string]any{"type": "number"},
											"targetIntensity":   map[string]any{"type": "string"},
										},
										"required": []string{"sport", "sessionType", "plannedDate"},
									},
								},
							},
							"required": []string{"weekNumber", "startDate", "sessions"},
						},
					},
				},
				"required": []string{"weeks"},
			}),
		},
		{
			Type:        "function",
			Name:        "archive_plan",
			Description: "Archive the active training plan (e.g. before creating a new one with a different focus).",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"planId": map[string]any{"type": "string"},
				},
				"required": []string{"planId"},
			}),
		},
		{
			Type:        "function",
			Name:        "get_activities",
			Description: "Read synced Garmin activities in a date range (max 31 days).",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []string{"from", "to"},
			}),
		},
		{
			Type:        "function",
			Name:        "get_daily_health",
			Description: "Read daily health metrics (sleep, HRV, resting HR, body battery, steps, weight) in a date range (max 31 days).",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []string{"from", "to"},
			}),
		},
		{
			Type:        "function",
			Name:        "get_plan_compliance",
			Description: "Read the active training plan with per-session compliance (completed, missed, planned).",
			Parameters: mustJSONSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}),
		},
		{
			Type:        "function",
			Name:        "get_climbing_log",
			Description: "Read the manual climbing log in a date range (max 31 days).",
			Parameters: mustJSONSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []string{"from", "to"},
			}),
		},
	}
}

func mustJSONSchema(schema map[string]any) json.RawMessage {
	raw, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return raw
}
