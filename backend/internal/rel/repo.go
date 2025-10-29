package rel

import (
	"context"

	databasepkg "github.com/Armour007/aura-backend/internal"
)

type TupleDB struct{}

func (TupleDB) Upsert(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	// naive: insert all
	for _, t := range tuples {
		_, err := databasepkg.DB.ExecContext(ctx, `INSERT INTO trust_tuples (object_type, object_id, relation, subject_type, subject_id, caveat_json) VALUES ($1,$2,$3,$4,$5,NULL)`, t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (TupleDB) Check(ctx context.Context, subjectType, subjectID, relation, objectType, objectID string) (bool, error) {
	var n int
	err := databasepkg.DB.GetContext(ctx, &n, `SELECT COUNT(1) FROM trust_tuples WHERE object_type=$1 AND object_id=$2 AND relation=$3 AND subject_type=$4 AND subject_id=$5`, objectType, objectID, relation, subjectType, subjectID)
	return n > 0, err
}
