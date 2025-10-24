package engine

import (
	"encoding/json"
	"fmt"
	"log" // For logging errors during evaluation

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

		// Simple MVP comparison: Check if requestMap 'contains' all key-value pairs from ruleMap's 'context'
		ruleContext, contextExists := ruleMap["context"].(map[string]interface{})
		ruleAction, actionExists := ruleMap["action"].(string)
		ruleEffect, effectExists := ruleMap["effect"].(string)

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
		if contextExists && len(ruleContext) > 0 {
			contextMatch = checkContextMatch(requestMap, ruleContext)
		}

		// If action and context match...
		if contextMatch {
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

// checkContextMatch performs a simple subset check: does requestData contain all key-value pairs from ruleData?
// NOTE: This is a basic implementation for MVP. Real-world needs more complex operators (>, <, wildcards etc.)
func checkContextMatch(requestData map[string]interface{}, ruleData map[string]interface{}) bool {
	for key, ruleValue := range ruleData {
		requestValue, ok := requestData[key]
		if !ok {
			return false // Key from rule is missing in request
		}
		// Basic equality check. Needs expansion for complex types/operators.
		if fmt.Sprintf("%v", requestValue) != fmt.Sprintf("%v", ruleValue) {
			return false // Values don't match
		}
	}
	return true // All rule keys/values were found and matched in request
}
