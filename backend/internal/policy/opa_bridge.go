//go:build disabled_by_default

package policy

import (
	"github.com/Armour007/aura-backend/internal/policy/opa"
)

func init() {
	opaNew = opa.OpaNew
}
