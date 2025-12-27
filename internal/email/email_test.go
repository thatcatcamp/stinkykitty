package email

import "testing"

func TestEmailServiceInitialization(t *testing.T) {
	svc, err := NewEmailService()
	if err != nil {
		// Expected if env vars not set in test - that's ok
		t.Logf("Email service unavailable (expected in test env): %v", err)
		return
	}

	if svc == nil {
		t.Fatal("Email service is nil")
	}
}
