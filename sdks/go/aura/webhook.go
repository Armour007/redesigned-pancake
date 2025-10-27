package aura

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

// ComputeSignature computes hex HMAC over "{ts}.{payload}".
func ComputeSignature(secret string, ts int64, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(strconv.FormatInt(ts, 10)))
	h.Write([]byte{"."[0]})
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature checks AURA-Signature: t=..., v1=...
func VerifySignature(secret, header string, payload []byte, tolerance time.Duration) (bool, error) {
	if header == "" {
		return false, errors.New("missing signature header")
	}
	parts := map[string]string{}
	for _, p := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}
	tsStr, ok := parts["t"]
	if !ok {
		return false, errors.New("missing t")
	}
	sigHex, ok := parts["v1"]
	if !ok {
		return false, errors.New("missing v1")
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false, err
	}
	if tolerance == 0 {
		tolerance = 5 * time.Minute
	}
	if time.Since(time.Unix(ts, 0)) > tolerance {
		return false, errors.New("timestamp expired")
	}
	expected := ComputeSignature(secret, ts, payload)
	a, _ := hex.DecodeString(expected)
	b, _ := hex.DecodeString(sigHex)
	if len(a) != len(b) {
		return false, nil
	}
	return hmac.Equal(a, b), nil
}
