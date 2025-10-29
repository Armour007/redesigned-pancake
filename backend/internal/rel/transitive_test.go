package rel

import (
	"context"
	"testing"
)

type memGraph struct{ edges []Tuple }

func (m *memGraph) Upsert(ctx context.Context, t Tuple) error {
	m.edges = append(m.edges, t)
	return nil
}
func (m *memGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error {
	m.edges = append(m.edges, tuples...)
	return nil
}
func (m *memGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	// simple BFS in-memory for tests: follow member/can_act_for, and allow final owner/editor/viewer implications
	type node struct {
		ns, id string
		depth  int
	}
	seen := map[string]bool{subject.Namespace + ":" + subject.ObjectID: true}
	q := []node{{subject.Namespace, subject.ObjectID, 0}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, e := range m.edges {
			if e.SubjectType == cur.ns && e.SubjectID == cur.id {
				if e.ObjectType == object.Namespace && e.ObjectID == object.ObjectID && implies(e.Relation, relation) {
					return true, "mem", nil
				}
				if e.Relation == "member" || e.Relation == "can_act_for" {
					k := e.ObjectType + ":" + e.ObjectID
					if !seen[k] {
						seen[k] = true
						q = append(q, node{e.ObjectType, e.ObjectID, cur.depth + 1})
					}
				}
			}
		}
	}
	return false, "mem", nil
}
func (m *memGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	return GraphExpansion{}, nil
}

func TestTransitiveImplications(t *testing.T) {
	mg := &memGraph{}
	// team devs member alice
	mg.Upsert(context.Background(), Tuple{ObjectType: "team", ObjectID: "devs", Relation: "member", SubjectType: "user", SubjectID: "alice"})
	// resource R1 editor team devs
	mg.Upsert(context.Background(), Tuple{ObjectType: "resource", ObjectID: "R1", Relation: "editor", SubjectType: "team", SubjectID: "devs"})
	// alice should be viewer via editor implication
	ok, _, _ := mg.Check(context.Background(), RelationRef{"user", "alice"}, "viewer", RelationRef{"resource", "R1"})
	if !ok {
		t.Fatal("expected viewer via editor->viewer implication and team membership")
	}
	// owner implies editor and viewer
	mg = &memGraph{}
	mg.Upsert(context.Background(), Tuple{ObjectType: "resource", ObjectID: "R2", Relation: "owner", SubjectType: "user", SubjectID: "alice"})
	if ok, _, _ := mg.Check(context.Background(), RelationRef{"user", "alice"}, "editor", RelationRef{"resource", "R2"}); !ok {
		t.Fatal("expected editor via owner implication")
	}
	if ok, _, _ := mg.Check(context.Background(), RelationRef{"user", "alice"}, "viewer", RelationRef{"resource", "R2"}); !ok {
		t.Fatal("expected viewer via owner implication")
	}
}
