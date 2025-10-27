package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ComputeWebhookSignature computes an HMAC-SHA256 signature over "{ts}.{payload}"
// similar to Stripe style signing. Returns hex string.
func ComputeWebhookSignature(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	msg := []byte(fmt.Sprintf("%d.", timestamp))
	msg = append(msg, payload...)
	mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature verifies provided hex signature given secret, timestamp, and payload
func VerifyWebhookSignature(secret string, timestamp int64, payload []byte, givenSigHex string) bool {
	expected := ComputeWebhookSignature(secret, timestamp, payload)
	exp, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	got, err := hex.DecodeString(givenSigHex)
	if err != nil {
		return false
	}
	return hmac.Equal(exp, got)
}
