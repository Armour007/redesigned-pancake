package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	databasepkg "github.com/Armour007/aura-backend/internal"
	"github.com/google/uuid"
)

// CreatePolicy inserts a policy row
func CreatePolicy(ctx context.Context, orgID uuid.UUID, name, engine string, createdBy *uuid.UUID) (databasepkg.Policy, error) {
	p := databasepkg.Policy{OrgID: orgID, Name: name, EngineType: engine, CreatedBy: createdBy}
	row := databasepkg.DB.QueryRowxContext(ctx, `INSERT INTO policies (org_id,name,engine_type,created_by_user_id) VALUES ($1,$2,$3,$4) RETURNING id, org_id, name, engine_type, created_by_user_id, created_at`, orgID, name, engine, createdBy)
	if err := row.StructScan(&p); err != nil {
		return databasepkg.Policy{}, err
	}
	return p, nil
}

// AddVersion inserts a policy version with auto-incremented version number per policy
func AddVersion(ctx context.Context, policyID uuid.UUID, body json.RawMessage, createdBy *uuid.UUID) (databasepkg.PolicyVersion, error) {
	var ver int
	if err := databasepkg.DB.GetContext(ctx, &ver, `SELECT COALESCE(MAX(version),0)+1 FROM policy_versions WHERE policy_id=$1`, policyID); err != nil {
		return databasepkg.PolicyVersion{}, err
	}
	pv := databasepkg.PolicyVersion{PolicyID: policyID, Version: ver, Body: body, CreatedBy: createdBy}
	row := databasepkg.DB.QueryRowxContext(ctx, `INSERT INTO policy_versions (policy_id,version,body,created_by_user_id,status) VALUES ($1,$2,$3,$4,'draft') RETURNING id, policy_id, version, body, compiled_blob, checksum, status, created_by_user_id, created_at, approved_by_user_id, approved_at, activated_at`, policyID, ver, body, createdBy)
	if err := row.StructScan(&pv); err != nil {
		return databasepkg.PolicyVersion{}, err
	}
	return pv, nil
}

func Assign(ctx context.Context, policyID uuid.UUID, scopeType, scopeID string) error {
	_, err := databasepkg.DB.ExecContext(ctx, `INSERT INTO policy_assignments (policy_id, scope_type, scope_id) VALUES ($1,$2,$3)`, policyID, scopeType, scopeID)
	return err
}

// GetActiveVersionForOrg fetches the latest active policy version assigned to org
func GetActiveVersionForOrg(ctx context.Context, orgID uuid.UUID) (*databasepkg.Policy, *databasepkg.PolicyVersion, error) {
	type row struct {
		PolicyID uuid.UUID `db:"policy_id"`
		Version  int       `db:"version"`
	}
	var r row
	err := databasepkg.DB.GetContext(ctx, &r, `SELECT pa.policy_id, pv.version FROM policy_assignments pa JOIN policy_versions pv ON pv.policy_id=pa.policy_id WHERE pa.scope_type='org' AND pa.scope_id=$1 AND pv.status='active' ORDER BY pv.version DESC LIMIT 1`, orgID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var p databasepkg.Policy
	if err := databasepkg.DB.GetContext(ctx, &p, `SELECT id, org_id, name, engine_type, created_by_user_id, created_at FROM policies WHERE id=$1`, r.PolicyID); err != nil {
		return nil, nil, err
	}
	var v databasepkg.PolicyVersion
	if err := databasepkg.DB.GetContext(ctx, &v, `SELECT id, policy_id, version, body, compiled_blob, checksum, status, created_by_user_id, created_at, approved_by_user_id, approved_at, activated_at FROM policy_versions WHERE policy_id=$1 AND version=$2`, r.PolicyID, r.Version); err != nil {
		return nil, nil, err
	}
	return &p, &v, nil
}

// GetActiveAssignmentsForOrg returns all active policy versions assigned to the org
func GetActiveAssignmentsForOrg(ctx context.Context, orgID uuid.UUID) ([]struct {
	Policy  databasepkg.Policy
	Version databasepkg.PolicyVersion
}, error) {
	rows := []struct {
		PolicyID uuid.UUID `db:"policy_id"`
		Version  int       `db:"version"`
	}{}
	if err := databasepkg.DB.SelectContext(ctx, &rows, `
		SELECT pa.policy_id, pv.version
		FROM policy_assignments pa
		JOIN policy_versions pv ON pv.policy_id=pa.policy_id
		WHERE pa.scope_type='org' AND pa.scope_id=$1 AND pv.status='active'
		ORDER BY pv.version DESC
	`, orgID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]struct {
		Policy  databasepkg.Policy
		Version databasepkg.PolicyVersion
	}, 0, len(rows))
	for _, r := range rows {
		var p databasepkg.Policy
		if err := databasepkg.DB.GetContext(ctx, &p, `SELECT id, org_id, name, engine_type, created_by_user_id, created_at FROM policies WHERE id=$1`, r.PolicyID); err != nil {
			return nil, err
		}
		var v databasepkg.PolicyVersion
		if err := databasepkg.DB.GetContext(ctx, &v, `SELECT id, policy_id, version, body, compiled_blob, checksum, status, created_by_user_id, created_at, approved_by_user_id, approved_at, activated_at FROM policy_versions WHERE policy_id=$1 AND version=$2`, r.PolicyID, r.Version); err != nil {
			return nil, err
		}
		out = append(out, struct {
			Policy  databasepkg.Policy
			Version databasepkg.PolicyVersion
		}{Policy: p, Version: v})
	}
	return out, nil
}

// GetPolicy returns a policy by id
func GetPolicy(ctx context.Context, id uuid.UUID) (databasepkg.Policy, error) {
	var p databasepkg.Policy
	err := databasepkg.DB.GetContext(ctx, &p, `SELECT id, org_id, name, engine_type, created_by_user_id, created_at FROM policies WHERE id=$1`, id)
	return p, err
}

// ActivateVersion sets the given policy version as active and marks others as draft
func ActivateVersion(ctx context.Context, policyID uuid.UUID, version int) error {
	tx, err := databasepkg.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Ensure the target version is approved before activation
	var status string
	if err := tx.GetContext(ctx, &status, `SELECT status FROM policy_versions WHERE policy_id=$1 AND version=$2`, policyID, version); err != nil {
		return err
	}
	if status != "approved" {
		return fmt.Errorf("cannot activate version %d: status must be 'approved' (got '%s')", version, status)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE policy_versions SET status='draft' WHERE policy_id=$1`, policyID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE policy_versions SET status='active', activated_at=NOW() WHERE policy_id=$1 AND version=$2`, policyID, version); err != nil {
		return err
	}
	return tx.Commit()
}

// ApproveVersion marks a version as approved with auditor information
func ApproveVersion(ctx context.Context, policyID uuid.UUID, version int, approvedBy *uuid.UUID) error {
	_, err := databasepkg.DB.ExecContext(ctx, `UPDATE policy_versions SET status='approved', approved_at=NOW(), approved_by_user_id=$3 WHERE policy_id=$1 AND version=$2`, policyID, version, approvedBy)
	return err
}

// RecordApproval stores an approval from a specific user for given policy version
func RecordApproval(ctx context.Context, policyID uuid.UUID, version int, user *uuid.UUID) error {
	if user == nil {
		return errors.New("user required for approval")
	}
	_, err := databasepkg.DB.ExecContext(ctx, `INSERT INTO policy_version_approvals(policy_id, version, user_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, policyID, version, *user)
	return err
}

// CountApprovals returns number of distinct approvers for a version
func CountApprovals(ctx context.Context, policyID uuid.UUID, version int) (int, error) {
	var n int
	err := databasepkg.DB.GetContext(ctx, &n, `SELECT COUNT(*) FROM policy_version_approvals WHERE policy_id=$1 AND version=$2`, policyID, version)
	return n, err
}

// GetVersion returns a specific version for a policy
func GetVersion(ctx context.Context, policyID uuid.UUID, version int) (databasepkg.PolicyVersion, error) {
	var v databasepkg.PolicyVersion
	err := databasepkg.DB.GetContext(ctx, &v, `SELECT id, policy_id, version, body, compiled_blob, checksum, status, created_by_user_id, created_at, approved_by_user_id, approved_at, activated_at FROM policy_versions WHERE policy_id=$1 AND version=$2`, policyID, version)
	return v, err
}
