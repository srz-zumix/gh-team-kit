package prgraph

import "testing"

func TestParseSubmodulePaths(t *testing.T) {
	content := []byte(`[submodule "LocalPackages/anjin-leap"]
	path = LocalPackages/anjin-leap
	url = https://github.com/example/anjin-leap.git
[submodule "quoted"]
	path = "LocalPackages/Sharin.Framework"
	branch = main
`)
	paths := parseSubmodulePaths(content)
	for _, want := range []string{"LocalPackages/anjin-leap", "LocalPackages/Sharin.Framework"} {
		if !paths[want] {
			t.Errorf("expected %q to be detected as a submodule, got %v", want, paths)
		}
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 submodule paths, got %v", paths)
	}
}

func TestPathNodeType(t *testing.T) {
	c := &collector{submodules: map[string]bool{"LocalPackages/anjin-leap": true}}
	tests := []struct {
		path   string
		folded bool
		want   NodeType
	}{
		{"LocalPackages/anjin-leap", false, NodeTypeSubmodule},
		{"LocalPackages/anjin-leap", true, NodeTypeDirectory},
		{"Assets/NPF/package.json", false, NodeTypeFile},
	}
	for _, tt := range tests {
		if got := c.pathNodeType(tt.path, tt.folded); got != tt.want {
			t.Errorf("pathNodeType(%q, %v) = %q, want %q", tt.path, tt.folded, got, tt.want)
		}
	}
}
