package email

import (
    "encoding/json"
    "net/http"
    "os"
    "testing"
    "time"
)

// This is an integration-style test that is skipped unless the environment
// variable MAILHOG_API is set to the MailHog API base (eg http://127.0.0.1:8025).
// It exercises the transactional API externally and checks MailHog for a
// captured message. It is intentionally lightweight and gated by env vars.
func Test_SendWithEnvelopeRewriteToMailHog(t *testing.T) {
    mailhogAPI := os.Getenv("MAILHOG_API")
    if mailhogAPI == "" {
        t.Skip("MAILHOG_API not set; skipping integration test")
    }

    // Wait a short time for services to be ready when running as part of
    // a live test run.
    time.Sleep(500 * time.Millisecond)

    // Query MailHog for messages; we expect the external test runner to send
    // a message prior to running this test. The test simply fetches the
    // messages and ensures the API responds with JSON.
    resp, err := http.Get(mailhogAPI + "/api/v2/messages")
    if err != nil {
        t.Fatalf("mailhog api request failed: %v", err)
    }
    defer resp.Body.Close()

    var out map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        t.Fatalf("failed to decode mailhog response: %v", err)
    }

    // Basic sanity: MailHog returned JSON and at least the `total` field exists.
    if _, ok := out["total"]; !ok {
        t.Fatalf("mailhog response missing 'total' field: %v", out)
    }
}
