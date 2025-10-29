//go:build spicedb

package rel

import (
	"context"

	authzedv1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
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
	_, err := s.client.WriteRelationships(ctx, &authzedv1.WriteRelationshipsRequest{
		Updates: []*authzedv1.RelationshipUpdate{{
			Operation: authzedv1.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &authzedv1.Relationship{
				Resource: &authzedv1.ObjectReference{ObjectType: t.ObjectType, ObjectId: t.ObjectID},
				Relation: t.Relation,
				Subject:  &authzedv1.SubjectReference{Object: &authzedv1.ObjectReference{ObjectType: t.SubjectType, ObjectId: t.SubjectID}},
			},
		}},
	})
	return err
}

func (s *SpiceDBGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error {
	ups := make([]*authzedv1.RelationshipUpdate, 0, len(tuples))
	for _, t := range tuples {
		ups = append(ups, &authzedv1.RelationshipUpdate{
			Operation: authzedv1.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &authzedv1.Relationship{
				Resource: &authzedv1.ObjectReference{ObjectType: t.ObjectType, ObjectId: t.ObjectID},
				Relation: t.Relation,
				Subject:  &authzedv1.SubjectReference{Object: &authzedv1.ObjectReference{ObjectType: t.SubjectType, ObjectId: t.SubjectID}},
			},
		})
	}
	_, err := s.client.WriteRelationships(ctx, &authzedv1.WriteRelationshipsRequest{Updates: ups})
	return err
}

func (s *SpiceDBGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	resp, err := s.client.CheckPermission(ctx, &authzedv1.CheckPermissionRequest{
		Resource:   &authzedv1.ObjectReference{ObjectType: object.Namespace, ObjectId: object.ObjectID},
		Permission: relation,
		Subject:    &authzedv1.SubjectReference{Object: &authzedv1.ObjectReference{ObjectType: subject.Namespace, ObjectId: subject.ObjectID}},
	})
	if err != nil {
		return false, "spicedb", err
	}
	return resp.GetPermissionship() == authzedv1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, "spicedb", nil
}

func (s *SpiceDBGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	// Simplified: we don't transform full expand tree; return root
	_, _ = s.client.ExpandPermissionTree(ctx, &authzedv1.ExpandPermissionTreeRequest{
		Resource:   &authzedv1.ObjectReference{ObjectType: object.Namespace, ObjectId: object.ObjectID},
		Permission: relation,
	})
	return GraphExpansion{Relation: relation, Object: object}, nil
}

var _ GraphClient = (*SpiceDBGraph)(nil)
