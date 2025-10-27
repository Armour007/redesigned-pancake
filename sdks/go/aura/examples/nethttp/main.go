package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Armour007/aura/sdks/go/aura"
)

func main() {
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		c := aura.NewClient(os.Getenv("AURA_API_KEY"), os.Getenv("AURA_API_BASE"), os.Getenv("AURA_VERSION"))
		resp, err := c.Verify(os.Getenv("AURA_AGENT_ID"), map[string]any{"action": "deploy:prod", "branch": "main"})
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/webhooks/aura", func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("AURA_WEBHOOK_SECRET")
		body, _ := io.ReadAll(r.Body)
		head := r.Header.Get("AURA-Signature")
		ok, _ := aura.VerifySignature(secret, head, body, 0)
		if !ok {
			w.WriteHeader(401)
			return
		}
		log.Println("webhook ok")
		w.WriteHeader(200)
	})

	log.Println("listening on :3002")
	log.Fatal(http.ListenAndServe(":3002", nil))
}
