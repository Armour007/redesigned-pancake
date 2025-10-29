package rel

import (
	"context"

	databasepkg "github.com/Armour007/aura-backend/internal"
)

// LocalGraph implements GraphClient using the trust_tuples SQL table with simple DFS
type LocalGraph struct{}

func NewLocalGraph() *LocalGraph { return &LocalGraph{} }

func (l *LocalGraph) Upsert(ctx context.Context, t Tuple) error {
	_, err := databasepkg.DB.ExecContext(ctx, `INSERT INTO trust_tuples (object_type, object_id, relation, subject_type, subject_id, caveat_json) VALUES ($1,$2,$3,$4,$5,NULL)`, t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID)
	return err
}

func (l *LocalGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error {
	for _, t := range tuples {
		if err := l.Upsert(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

// Check supports direct edge and a limited 3-hop transitive traversal via member/editor/viewer/owner can_act_for
func (l *LocalGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	// direct relation (including relation implication)
	if ok, err := l.hasDirectOrImplied(ctx, subject, relation, object); err != nil {
		return false, "local", err
	} else if ok {
		return true, "local", nil
	}

	// BFS up to depth 3 following membership/delegation edges to intermediate nodes
	type node struct {
		ns, id string
		depth  int
	}
	q := []node{{ns: subject.Namespace, id: subject.ObjectID, depth: 0}}
	seen := map[string]bool{subject.Namespace + ":" + subject.ObjectID: true}
	maxDepth := 3
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.depth >= maxDepth {
			continue
		}
		rows, err := databasepkg.DB.QueryxContext(ctx, `SELECT object_type, object_id, relation FROM trust_tuples WHERE subject_type=$1 AND subject_id=$2`, cur.ns, cur.id)
		if err != nil {
			return false, "local", err
		}
		for rows.Next() {
			var ot, oid, rel string
			if err := rows.Scan(&ot, &oid, &rel); err != nil {
				_ = rows.Close()
				return false, "local", err
			}
			// final hop: does this edge grant or imply the desired relation on the target object?
			if ot == object.Namespace && oid == object.ObjectID && implies(rel, relation) {
				_ = rows.Close()
				return true, "local", nil
			}
			// intermediate hop: traverse only through membership/delegation
			if rel == "member" || rel == "can_act_for" {
				key := ot + ":" + oid
				if !seen[key] {
					seen[key] = true
					q = append(q, node{ns: ot, id: oid, depth: cur.depth + 1})
				}
			}
		}
		_ = rows.Close()
	}
	return false, "local", nil
}

func (l *LocalGraph) hasDirectOrImplied(ctx context.Context, subject RelationRef, desired string, object RelationRef) (bool, error) {
	// check for any tuple that satisfies desired via implication at target
	// e.g., owner implies editor implies viewer
	query := `SELECT relation FROM trust_tuples WHERE object_type=$1 AND object_id=$2 AND subject_type=$3 AND subject_id=$4`
	rows, err := databasepkg.DB.QueryxContext(ctx, query, object.Namespace, object.ObjectID, subject.Namespace, subject.ObjectID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var rel string
		if err := rows.Scan(&rel); err != nil {
			return false, err
		}
		if implies(rel, desired) {
			return true, nil
		}
	}
	return false, nil
}

func implies(have, want string) bool {
	if have == want {
		return true
	}
	// owner -> editor -> viewer
	if have == "owner" && (want == "editor" || want == "viewer") {
		return true
	}
	if have == "editor" && want == "viewer" {
		return true
	}
	return false
}

func (l *LocalGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	if depth <= 0 {
		return GraphExpansion{Relation: relation, Object: object}, nil
	}
	// list direct subjects
	rows, err := databasepkg.DB.QueryxContext(ctx, `SELECT subject_type, subject_id FROM trust_tuples WHERE object_type=$1 AND object_id=$2 AND relation=$3`, object.Namespace, object.ObjectID, relation)
	if err != nil {
		return GraphExpansion{}, err
	}
	defer rows.Close()
	exp := GraphExpansion{Relation: relation, Object: object}
	for rows.Next() {
		var st, sid string
		if err := rows.Scan(&st, &sid); err != nil {
			return GraphExpansion{}, err
		}
		child := GraphExpansion{Relation: "subject", Object: RelationRef{Namespace: st, ObjectID: sid}}
		exp.Children = append(exp.Children, child)
	}
	return exp, nil
}

var _ GraphClient = (*LocalGraph)(nil)
