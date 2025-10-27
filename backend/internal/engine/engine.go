package engine

import (
	"encoding/json"
	"fmt"
	"log" // For logging errors during evaluation
	"strconv"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal" // Adjust path
	"github.com/google/uuid"
	// We need a library to easily compare JSON/maps
	// Using a simple comparison for MVP, could use JSONPath or similar later
)

// Evaluate checks if the request context matches any allow rule for the agent.
// Returns true if allowed, false otherwise, and a reason string.
func Evaluate(agentID uuid.UUID, requestContext json.RawMessage) (bool, string) {
	// 1. Fetch Rules from Database
	var rules []database.Permission
	query := `SELECT rule FROM permissions WHERE agent_id = $1 AND is_active = true ORDER BY created_at ASC` // Get only active rules
	err := database.DB.Select(&rules, query, agentID)
	if err != nil {
		log.Printf("Error fetching rules for agent %s: %v", agentID, err)
		return false, "Internal error fetching rules" // Deny if rules can't be fetched
	}

	if len(rules) == 0 {
		return false, "No active rules defined for this agent" // Deny if no rules exist
	}

	// 2. Unmarshal the Request Context into a map for easier comparison
	var requestMap map[string]interface{}
	if err := json.Unmarshal(requestContext, &requestMap); err != nil {
		log.Printf("Error unmarshalling request context for agent %s: %v", agentID, err)
		return false, "Invalid request context format"
	}

	// 3. Evaluate Rules (Deny by Default)
	decision := false // Start with DENIED
	reason := "No matching allow rule found"

	for _, permission := range rules {
		// Unmarshal the stored rule JSON
		var ruleMap map[string]interface{}
		if err := json.Unmarshal(permission.Rule, &ruleMap); err != nil {
			log.Printf("Error unmarshalling stored rule ID %s for agent %s: %v", permission.ID, agentID, err)
			continue // Skip invalid rules
		}

		// Extract core fields
		ruleAction, actionExists := ruleMap["action"].(string)
		ruleEffectRaw, effectExists := ruleMap["effect"].(string)
		ruleEffect := strings.ToLower(ruleEffectRaw)
		ruleContext, _ := ruleMap["context"].(map[string]interface{})
		// Optional time window constraint
		var timeWindow map[string]interface{}
		if tw, ok := ruleMap["time_window"].(map[string]interface{}); ok {
			timeWindow = tw
		}

		// Basic validation of the rule structure
		if !actionExists || !effectExists {
			log.Printf("Skipping invalid rule structure (missing action/effect) ID %s for agent %s", permission.ID, agentID)
			continue
		}

		// Check if action matches
		requestAction, requestActionExists := requestMap["action"].(string)
		if !requestActionExists || requestAction != ruleAction {
			continue // Actions don't match, try next rule
		}

		// Check if context matches (if rule has context)
		contextMatch := true // Assume match if no context defined in rule
		if len(ruleContext) > 0 {
			contextMatch = checkContextMatch(requestMap, ruleContext)
		}

		// Evaluate time window if present
		timeMatch := true
		if timeWindow != nil {
			timeMatch = checkTimeWindow(timeWindow)
		}

		// If action and context/time match...
		if contextMatch && timeMatch {
			if ruleEffect == "allow" {
				decision = true // Found an ALLOW rule that matches!
				reason = "Request matched allow rule"
				break // Stop processing on first allow match
			} else if ruleEffect == "deny" {
				decision = false // Found a DENY rule that matches!
				reason = "Request matched deny rule"
				break // Explicit deny takes precedence, stop processing
			}
		}
	}

	return decision, reason
}

// checkContextMatch evaluates rule context with basic operators and AND support.
// Supported forms:
// - { "branch": { "eq": "main" } }
// - { "AND": [ { "env": {"eq":"prod"}}, {"version": {"gte": 2}} ] }
// - Legacy shorthand: { "env": "prod" }
func checkContextMatch(requestData map[string]interface{}, ruleData map[string]interface{}) bool {
	// AND support
	if andRaw, ok := ruleData["AND"]; ok {
		andSlice, ok := andRaw.([]interface{})
		if !ok {
			return false
		}
		for _, item := range andSlice {
			m, ok := item.(map[string]interface{})
			if !ok || !checkContextMatch(requestData, m) {
				return false
			}
		}
		return true
	}

	// Key-operator-value pairs
	for key, rv := range ruleData {
		// Skip reserved keys
		if key == "AND" || key == "OR" {
			continue
		}
		reqVal, ok := requestData[key]
		if !ok {
			return false
		}
		switch cond := rv.(type) {
		case map[string]interface{}:
			if !evalOperators(reqVal, cond) {
				return false
			}
		default:
			// Legacy equality
			if fmt.Sprintf("%v", reqVal) != fmt.Sprintf("%v", cond) {
				return false
			}
		}
	}
	return true
}

// evalOperators evaluates a set of operators against a single request value.
// Supported ops: eq, neq, gt, gte, lt, lte, contains
func evalOperators(req interface{}, ops map[string]interface{}) bool {
	for op, v := range ops {
		switch strings.ToLower(op) {
		case "eq":
			if fmt.Sprintf("%v", req) != fmt.Sprintf("%v", v) {
				return false
			}
		case "neq":
			if fmt.Sprintf("%v", req) == fmt.Sprintf("%v", v) {
				return false
			}
		case "gt":
			if !compareNumber(req, v, ">") {
				return false
			}
		case "gte":
			if !compareNumber(req, v, ">=") {
				return false
			}
		case "lt":
			if !compareNumber(req, v, "<") {
				return false
			}
		case "lte":
			if !compareNumber(req, v, "<=") {
				return false
			}
		case "contains":
			rs := fmt.Sprintf("%v", req)
			vs := fmt.Sprintf("%v", v)
			if !strings.Contains(rs, vs) {
				return false
			}
		default:
			// Unknown operator -> fail safe
			return false
		}
	}
	return true
}

func compareNumber(a interface{}, b interface{}, op string) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if !aok || !bok {
		return false
	}
	switch op {
	case ">":
		return af > bf
	case ">=":
		return af >= bf
	case "<":
		return af < bf
	case "<=":
		return af <= bf
	default:
		return false
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		// Try to parse string to float
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// checkTimeWindow validates if current time lies within the configured window.
// { "days": ["Mon","Tue","Wed","Thu","Fri"], "start": "09:00", "end": "18:00", "tz": "Asia/Kolkata" }
func checkTimeWindow(tw map[string]interface{}) bool {
	// Timezone
	loc := time.UTC
	if tzRaw, ok := tw["tz"].(string); ok && tzRaw != "" {
		if l, err := time.LoadLocation(tzRaw); err == nil {
			loc = l
		}
	}

	now := time.Now().In(loc)
	// Day match
	dayOK := true
	if daysRaw, ok := tw["days"].([]interface{}); ok && len(daysRaw) > 0 {
		dayOK = false
		wd := now.Weekday().String()[:3] // e.g., Mon
		for _, d := range daysRaw {
			if ds, ok := d.(string); ok && strings.EqualFold(ds[:3], wd) {
				dayOK = true
				break
			}
		}
	}

	if !dayOK {
		return false
	}

	// Time range
	startOK, endOK := false, false
	var startMin, endMin int
	if s, ok := tw["start"].(string); ok {
		if m, ok := parseHHMM(s); ok {
			startMin = m
			startOK = true
		}
	}
	if e, ok := tw["end"].(string); ok {
		if m, ok := parseHHMM(e); ok {
			endMin = m
			endOK = true
		}
	}
	if !startOK || !endOK {
		return true
	} // If not configured properly, ignore constraint

	curMin := now.Hour()*60 + now.Minute()
	return curMin >= startMin && curMin <= endMin
}

func parseHHMM(s string) (int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
