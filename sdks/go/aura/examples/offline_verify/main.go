package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Armour007/aura/sdks/go/aura"
)

func main() {
	base := os.Getenv("AURA_API_BASE_URL")
	if base == "" {
		base = "http://localhost:8081"
	}
	org := os.Getenv("AURA_ORG_ID")
	token := os.Getenv("AURA_TRUST_TOKEN")
	if token == "" {
		fmt.Println("Set AURA_TRUST_TOKEN to a JWT to verify")
		os.Exit(2)
	}

	ctx := context.Background()
	cache := aura.NewTrustCache(5*time.Minute, 1*time.Minute)
	res, err := aura.VerifyTrustTokenOfflineCached(ctx, cache, base, token, org, 10)
	if err != nil {
		panic(err)
	}
	if !res.Valid {
		fmt.Printf("invalid: %s\n", res.Reason)
		os.Exit(1)
	}
	fmt.Printf("valid. claims: %+v\n", res.Claims)
}
