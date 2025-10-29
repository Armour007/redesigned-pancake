package rel

import "sync"

// Tuple represents (object, relation, subject)
type Tuple struct {
	ObjectType  string `json:"object_type"`
	ObjectID    string `json:"object_id"`
	Relation    string `json:"relation"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
}

// Store is an in-memory tuple store for prototype
type Store struct {
	mu     sync.RWMutex
	tuples []Tuple
}

func NewStore() *Store { return &Store{tuples: []Tuple{}} }

func (s *Store) Upsert(ts []Tuple) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// simplistic: append; no de-dup
	s.tuples = append(s.tuples, ts...)
}

func (s *Store) Check(subjectType, subjectID, relation, objectType, objectID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tuples {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation && t.SubjectType == subjectType && t.SubjectID == subjectID {
			return true
		}
	}
	return false
}
