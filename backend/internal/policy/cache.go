package policy

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// simple compiled policy cache keyed by policyID:version
var compiledCache sync.Map // key string -> CompiledPolicy

func cacheKey(pid uuid.UUID, version int) string { return fmt.Sprintf("%s:%d", pid.String(), version) }

func GetCompiled(pid uuid.UUID, version int) (CompiledPolicy, bool) {
	if v, ok := compiledCache.Load(cacheKey(pid, version)); ok {
		if cp, ok2 := v.(CompiledPolicy); ok2 {
			return cp, true
		}
	}
	return nil, false
}

func PutCompiled(pid uuid.UUID, version int, cp CompiledPolicy) {
	compiledCache.Store(cacheKey(pid, version), cp)
}

// DeleteCompiled removes a compiled entry; if version <= 0, remove all versions for the policy
func DeleteCompiled(pid uuid.UUID, version int) {
	if version > 0 {
		compiledCache.Delete(cacheKey(pid, version))
		return
	}
	// range and delete all keys matching pid prefix
	prefix := pid.String() + ":"
	compiledCache.Range(func(k, _ any) bool {
		if ks, ok := k.(string); ok && len(ks) >= len(prefix) && ks[:len(prefix)] == prefix {
			compiledCache.Delete(k)
		}
		return true
	})
}
