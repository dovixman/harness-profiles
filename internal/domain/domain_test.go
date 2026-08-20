package domain

import (
	"strings"
	"testing"
)

func TestValidateHarnessID_RejectsInvalidNames(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "blank", id: " ", want: "required"},
		{name: "separator", id: "bad/name", want: "path separators"},
		{name: "reserved dot", id: ".", want: "reserved"},
		{name: "reserved dotdot", id: "..", want: "reserved"},
		{name: "whitespace", id: "bad name", want: "whitespace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHarnessID(tt.id)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHarnessID(%q) = %v, want error containing %q", tt.id, err, tt.want)
			}
		})
	}
}

func TestConfigValidate_RejectsDuplicatesCaseInsensitively(t *testing.T) {
	tests := []struct {
		name      string
		harnesses []Harness
		want      string
	}{
		{
			name: "duplicate id",
			harnesses: []Harness{
				{ID: "claude", Label: "Claude", LinkPath: "/tmp/claude"},
				{ID: "CLAUDE", Label: "Other", LinkPath: "/tmp/other"},
			},
			want: "duplicate harness id",
		},
		{
			name: "duplicate label",
			harnesses: []Harness{
				{ID: "claude", Label: "Claude", LinkPath: "/tmp/claude"},
				{ID: "opencode", Label: " claude ", LinkPath: "/tmp/opencode"},
			},
			want: "duplicate harness label",
		},
		{
			name: "duplicate link",
			harnesses: []Harness{
				{ID: "claude", Label: "Claude", LinkPath: "/tmp/root"},
				{ID: "opencode", Label: "OpenCode", LinkPath: " /tmp/root "},
			},
			want: "duplicate harness root path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Config{HarnessesRoot: "/tmp/harnesses", Harnesses: tt.harnesses}).Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestConfigValidate_RejectsDuplicateLinkPathsAcrossHarnesses(t *testing.T) {
	tests := []struct {
		name      string
		harnesses []Harness
		want      string
	}{
		{
			name: "duplicate legacy path with whitespace normalization",
			harnesses: []Harness{
				{ID: "claude", Label: "Claude", LinkPath: " /tmp/root "},
				{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/root/../root"},
			},
			want: "duplicate harness root path",
		},
		{
			name: "duplicate multi-link path",
			harnesses: []Harness{
				{ID: "claude", Label: "Claude", Links: []HarnessLink{{ID: "root", Path: "/tmp/agents", Kind: HarnessLinkKindDir}}},
				{ID: "opencode", Label: "OpenCode", Links: []HarnessLink{{ID: "root", Path: "/tmp/agents/../agents", Kind: HarnessLinkKindDir}}},
			},
			want: "duplicate harness root path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Config{HarnessesRoot: "/tmp/harnesses", Harnesses: tt.harnesses}).Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestConfigValidate_AcceptsValidConfig(t *testing.T) {
	config := Config{HarnessesRoot: "/tmp/harnesses", Harnesses: []Harness{{ID: "claude", Label: "Claude", LinkPath: "/tmp/claude"}}}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestHarnessValidate_RequiresLabelAndLinkData(t *testing.T) {
	tests := []struct {
		name    string
		harness Harness
		want    string
	}{
		{name: "label", harness: Harness{ID: "claude", LinkPath: "/tmp/root"}, want: "label is required"},
		{name: "links invalid id", harness: Harness{ID: "claude", Label: "Claude", Links: []HarnessLink{{ID: "bad name", Path: "/tmp/root", Kind: HarnessLinkKindDir}}}, want: "whitespace"},
		{name: "links invalid kind", harness: Harness{ID: "claude", Label: "Claude", Links: []HarnessLink{{ID: "root", Path: "/tmp/root", Kind: "bogus"}}}, want: "invalid"},
		{name: "links missing path", harness: Harness{ID: "claude", Label: "Claude", Links: []HarnessLink{{ID: "root", Path: "", Kind: HarnessLinkKindDir}}}, want: "required"},
		{name: "links duplicate id", harness: Harness{ID: "claude", Label: "Claude", Links: []HarnessLink{{ID: "root", Path: "/tmp/root", Kind: HarnessLinkKindDir}, {ID: "ROOT", Path: "/tmp/other", Kind: HarnessLinkKindDir}}}, want: "duplicate harness link id"},
		{name: "link", harness: Harness{ID: "claude", Label: "Claude"}, want: "link path is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.harness.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestHarnessValidate_RejectsDuplicateLinkPaths(t *testing.T) {
	harness := Harness{
		ID:    "claude",
		Label: "Claude",
		Links: []HarnessLink{
			{ID: "root", Path: "/tmp/root", Kind: HarnessLinkKindDir},
			{ID: "state", Path: "/tmp/root/../root", Kind: HarnessLinkKindDir},
		},
	}

	if err := harness.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate harness link path") {
		t.Fatalf("Validate() = %v, want duplicate link path error", err)
	}
}

func TestHarnessLinksOrLegacy(t *testing.T) {
	legacy := Harness{ID: "claude", Label: "Claude", LinkPath: "/tmp/root"}
	if got := legacy.LinksOrLegacy(); len(got) != 1 {
		t.Fatalf("LinksOrLegacy() length = %d, want 1", len(got))
	}
	if got := legacy.LinksOrLegacy()[0]; got.ID != LegacyDefaultLinkID || got.Kind != HarnessLinkKindDir || got.Path != "/tmp/root" {
		t.Fatalf("LinksOrLegacy() = %#v, want id=%q kind=%q path=%q", got, LegacyDefaultLinkID, HarnessLinkKindDir, "/tmp/root")
	}

	multi := Harness{
		ID:       "claude",
		Label:    "Claude",
		LinkPath: "/tmp/legacy",
		Links: []HarnessLink{{
			ID:   "agents",
			Path: "/tmp/agents",
			Kind: HarnessLinkKindDir,
		}},
	}
	if got := multi.LinksOrLegacy(); len(got) != 1 || got[0].ID != "agents" {
		t.Fatalf("LinksOrLegacy() = %#v, want single agents link", got)
	}
}

func TestValidateProfileName_ReusesNameRules(t *testing.T) {
	if err := ValidateProfileName("work"); err != nil {
		t.Fatalf("ValidateProfileName(valid) = %v", err)
	}
	if err := ValidateProfileName("bad/name"); err == nil {
		t.Fatal("ValidateProfileName(invalid) = nil, want error")
	}
}

func TestDeleteModeValidate(t *testing.T) {
	mode, err := ParseDeleteMode("restore")
	if err != nil || mode != DeleteModeRestore {
		t.Fatalf("ParseDeleteMode() = %q, %v", mode, err)
	}
	if err := DeleteMode("bad").Validate(); err == nil {
		t.Fatal("Validate() = nil, want error")
	}
}
