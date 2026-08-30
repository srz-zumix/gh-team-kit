package prgraph

import (
	"reflect"
	"testing"

	ignore "github.com/sabhiram/go-gitignore"
)

func TestParseGeneratedPatterns(t *testing.T) {
	content := []byte(`# comment
*.pb.go linguist-generated
docs/api.md linguist-generated=true
vendor/** linguist-generated -diff
"Assets/Generated Code/*.cs" linguist-generated
src/keep.pb.go -linguist-generated
*.md linguist-documentation
untouched
`)
	want := []string{
		"*.pb.go",
		"docs/api.md",
		"vendor/**",
		"Assets/Generated Code/*.cs",
		"!src/keep.pb.go",
	}
	if got := parseGeneratedPatterns(content); !reflect.DeepEqual(got, want) {
		t.Errorf("parseGeneratedPatterns() = %v, want %v", got, want)
	}
}

func TestIsGeneratedFile(t *testing.T) {
	c := &collector{generatedFiles: ignore.CompileIgnoreLines(parseGeneratedPatterns([]byte(
		"*.pb.go linguist-generated\nsrc/keep.pb.go -linguist-generated\n"))...)}
	tests := []struct {
		filename string
		want     bool
	}{
		{"api/service.pb.go", true},
		{"src/keep.pb.go", false},
		{"main.go", false},
	}
	for _, tt := range tests {
		if got := c.isGeneratedFile(tt.filename); got != tt.want {
			t.Errorf("isGeneratedFile(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestIsGeneratedFileWithoutMatcher(t *testing.T) {
	c := &collector{}
	if c.isGeneratedFile("api/service.pb.go") {
		t.Error("expected no file to be generated without --exclude-generated")
	}
}
