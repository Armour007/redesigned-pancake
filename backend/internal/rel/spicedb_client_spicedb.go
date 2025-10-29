//go:build spicedb

package rel

import (
	"context"

	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/authzed-go/v1/authzed"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SpiceDBGraph implements GraphClient using authzed-go. Build with -tags spicedb
type SpiceDBGraph struct {
	client authzed.Client
}

func NewSpiceDBGraph(endpoint, token string) (*SpiceDBGraph, error) {
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c := authzed.NewClientWithConn(conn, authzed.WithToken(token))
	return &SpiceDBGraph{client: c}, nil
}

func (s *SpiceDBGraph) withToken(ctx context.Context) context.Context { return ctx }

func (s *SpiceDBGraph) Upsert(ctx context.Context, t Tuple) error {
	_, err := s.client.WriteRelationships(ctx, &authzed.WriteRelationshipsRequest{
		Updates: []*authzed.RelationshipUpdate{{
			Operation: authzed.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &authzed.Relationship{
				Resource: &authzed.ObjectReference{ObjectType: t.ObjectType, ObjectId: t.ObjectID},
				Relation: t.Relation,
				Subject:  &authzed.SubjectReference{Object: &authzed.ObjectReference{ObjectType: t.SubjectType, ObjectId: t.SubjectID}},
			},
		}},
	})
	return err
}

func (s *SpiceDBGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error {
	ups := make([]*authzed.RelationshipUpdate, 0, len(tuples))
	for _, t := range tuples {
		ups = append(ups, &authzed.RelationshipUpdate{
			Operation: authzed.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &authzed.Relationship{
				Resource: &authzed.ObjectReference{ObjectType: t.ObjectType, ObjectId: t.ObjectID},
				Relation: t.Relation,
				Subject:  &authzed.SubjectReference{Object: &authzed.ObjectReference{ObjectType: t.SubjectType, ObjectId: t.SubjectID}},
			},
		})
	}
	_, err := s.client.WriteRelationships(ctx, &authzed.WriteRelationshipsRequest{Updates: ups})
	return err
}

func (s *SpiceDBGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	resp, err := s.client.CheckPermission(ctx, &authzed.CheckPermissionRequest{
		Resource:   &authzed.ObjectReference{ObjectType: object.Namespace, ObjectId: object.ObjectID},
		Permission: relation,
		Subject:    &authzed.SubjectReference{Object: &authzed.ObjectReference{ObjectType: subject.Namespace, ObjectId: subject.ObjectID}},
	})
	if err != nil {
		return false, "spicedb", err
	}
	return resp.GetPermissionship() == authzed.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, "spicedb", nil
}

func (s *SpiceDBGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	// Simplified: we don't transform full expand tree; return root
	_, _ = s.client.ExpandPermissionTree(ctx, &authzed.ExpandPermissionTreeRequest{
		Resource:   &authzed.ObjectReference{ObjectType: object.Namespace, ObjectId: object.ObjectID},
		Permission: relation,
	})
	return GraphExpansion{Relation: relation, Object: object}, nil
}

var _ GraphClient = (*SpiceDBGraph)(nil)
