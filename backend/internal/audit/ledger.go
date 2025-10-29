package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	db "github.com/Armour007/aura-backend/internal"
	"github.com/google/uuid"
)

// Append writes a new audit event with a hash chain per org.
// this_hash = SHA256(prev_hash_bytes || canonical_json)
func Append(ctx context.Context, orgID uuid.UUID, eventType string, payload any, actorUser *uuid.UUID, actorAgent *uuid.UUID) error {
	// get last hash for org (by seq desc)
	var prev string
	_ = db.DB.GetContext(ctx, &prev, `SELECT this_hash FROM audit_ledger WHERE org_id=$1 ORDER BY seq DESC LIMIT 1`, orgID)
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h := sha256.New()
	if prev != "" {
		pb, _ := hex.DecodeString(prev)
		h.Write(pb)
	}
	h.Write(b)
	sum := h.Sum(nil)
	hs := hex.EncodeToString(sum)
	_, err = db.DB.ExecContext(ctx, `INSERT INTO audit_ledger(org_id, actor_user_id, actor_agent_id, event_type, payload, prev_hash, this_hash) VALUES ($1,$2,$3,$4,$5,$6,$7)`, orgID, actorUser, actorAgent, eventType, b, prev, hs)
	return err
}

// Verify walks the chain for org and returns first break index (1-based seq) or 0 when OK.
func Verify(ctx context.Context, orgID uuid.UUID, limit int) (int64, error) {
	type row struct {
		Seq     int64  `db:"seq"`
		Prev    string `db:"prev_hash"`
		This    string `db:"this_hash"`
		Payload []byte `db:"payload"`
	}
	rows := []row{}
	if limit <= 0 || limit > 10000 {
		limit = 10000
	}
	if err := db.DB.SelectContext(ctx, &rows, `SELECT seq, prev_hash, this_hash, payload FROM audit_ledger WHERE org_id=$1 ORDER BY seq ASC LIMIT $2`, orgID, limit); err != nil {
		return 0, err
	}
	var last string
	for _, r := range rows {
		h := sha256.New()
		if last != "" {
			pb, _ := hex.DecodeString(last)
			h.Write(pb)
		}
		h.Write(r.Payload)
		sum := h.Sum(nil)
		if hex.EncodeToString(sum) != r.This {
			return r.Seq, fmt.Errorf("hash mismatch at seq %d", r.Seq)
		}
		last = r.This
	}
	return 0, nil
}
