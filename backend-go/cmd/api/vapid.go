package main

import (
	"fmt"
	"io"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// vapidKeys is `api vapid-keys`: prints a fresh Web Push key pair as the two .env lines the reminder sender reads
// (internal/care). Run it once per environment; changing the keys later invalidates every stored subscription.
func vapidKeys(w io.Writer) int {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		_, _ = fmt.Fprintln(w, "vapid-keys:", err)
		return 1
	}
	_, _ = fmt.Fprintf(w, "VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", public, private)
	return 0
}
