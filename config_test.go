package main

import "testing"

func TestLoadConfigUsesWeComBotIDForGroupTriggerMentionFallback(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WECOM_CORP_ID", "corp")
	t.Setenv("WECOM_CORP_SECRET", "secret")
	t.Setenv("WECOM_RSA_PRIVATE_KEY", "private-key")
	t.Setenv("WECOM_BOT_ID", "moss")
	t.Setenv("WECOM_GROUP_TRIGGER_MENTIONS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.WeComBotID != "moss" {
		t.Fatalf("WeComBotID = %q, want moss", cfg.WeComBotID)
	}
	if len(cfg.WeComGroupTriggerMentions) != 1 || cfg.WeComGroupTriggerMentions[0] != "moss" {
		t.Fatalf("WeComGroupTriggerMentions = %#v, want [moss]", cfg.WeComGroupTriggerMentions)
	}
}

func TestLoadConfigUsesDefaultSandboxWakePlaceholder(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.SandboxWakePlaceholder != defaultSandboxWakePlaceholder {
		t.Fatalf("SandboxWakePlaceholder = %q, want %q", cfg.SandboxWakePlaceholder, defaultSandboxWakePlaceholder)
	}
}

func TestLoadConfigCanDisableSandboxWakePlaceholder(t *testing.T) {
	t.Setenv("SANDBOX_WAKE_PLACEHOLDER", "off")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.SandboxWakePlaceholder != "" {
		t.Fatalf("SandboxWakePlaceholder = %q, want empty", cfg.SandboxWakePlaceholder)
	}
}
