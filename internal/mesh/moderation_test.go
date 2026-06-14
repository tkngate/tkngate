package mesh

import (
	"testing"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

func TestCheckModeration_Disabled(t *testing.T) {
	logging.InitLogger()
	config.Cfg = config.Config{}
	config.Cfg.Mesh.ReputationEnabled = false

	safe, err := CheckModeration("kill everyone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !safe {
		t.Error("expected safe=true when moderation is disabled")
	}
}

func TestCheckModeration_NoAPIKey(t *testing.T) {
	logging.InitLogger()
	config.Cfg = config.Config{}
	config.Cfg.Mesh.ReputationEnabled = true
	config.Cfg.Mesh.PreflightModeration = true
	config.Cfg.Mesh.ModerationAPIKey = ""

	safe, err := CheckModeration("some prompt")
	if err == nil {
		t.Fatalf("expected error when no api key is configured and fail-closed is active")
	}
	if safe {
		t.Error("expected fail-closed safe=false when no api key is configured")
	}
}

// In a real test suite, we would mock the http.Client used in CheckModeration,
// but since this is an MVP we'll just test the disabled and no-key paths.
// Testing the live OpenAI API requires a valid API key.
