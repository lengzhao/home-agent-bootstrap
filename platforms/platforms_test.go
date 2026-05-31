package platforms

import "testing"

func TestParseChoicesSingleDefault(t *testing.T) {
	got, err := ParseChoices("7")
	if err != nil {
		t.Fatalf("ParseChoices() error: %v", err)
	}
	if len(got) != 1 || got[0].Type != "weixin" {
		t.Fatalf("ParseChoices(7) = %#v", got)
	}
}

func TestParseChoicesMultiple(t *testing.T) {
	got, err := ParseChoices("1,7")
	if err != nil {
		t.Fatalf("ParseChoices() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ParseChoices(1,7) len = %d, want 2", len(got))
	}
}

func TestCountWeixin(t *testing.T) {
	blocks := []Block{
		{Type: "weixin"},
		{Type: "telegram"},
		{Type: "weixin"},
	}
	if got := CountWeixin(blocks); got != 2 {
		t.Fatalf("CountWeixin() = %d, want 2", got)
	}
}

func TestSetupHintsSkipsWeixin(t *testing.T) {
	blocks := []Block{
		{Type: "weixin"},
		{Type: "feishu"},
	}
	hints := SetupHints(blocks)
	if len(hints) != 1 {
		t.Fatalf("SetupHints() = %#v, want one feishu hint", hints)
	}
}
