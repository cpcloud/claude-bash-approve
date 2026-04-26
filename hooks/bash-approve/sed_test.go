package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/syntax"
)

// parseCallArgs parses a single bash command string and returns the AST args
// of the resulting CallExpr. Reused by sed/awk/tee/env tests in this package.
func parseCallArgs(t *testing.T, cmd string) []*syntax.Word {
	t.Helper()
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(strings.NewReader(cmd), "")
	require.NoError(t, err)
	call := file.Stmts[0].Cmd.(*syntax.CallExpr)
	return call.Args
}

func TestIsSedSafe(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		wantOverride bool
	}{
		{"plain substitution", "sed 's/foo/bar/' file", false},
		{"print range", "sed -n '1,5p' file", false},
		{"in-place short", "sed -i 's/foo/bar/' file", true},
		{"in-place backup", "sed -i.bak 's/foo/bar/' file", true},
		{"in-place long", "sed --in-place 's/foo/bar/' file", true},
		{"-e expression", "sed -e 's/a/b/' -e 's/c/d/' file", false},
		{"-f script", "sed -f script.sed file", false},
		{"escaped in-place short", `sed \-i 's/foo/bar/' file`, true},
		{"escaped in-place long", `sed \--in-place 's/foo/bar/' file`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := parseCallArgs(t, tt.cmd)
			ok := isSedSafe(args, evalContext{})
			if tt.wantOverride {
				assert.False(t, ok, "expected validator to reject %q", tt.cmd)
			} else {
				assert.True(t, ok, "expected validator to accept %q", tt.cmd)
			}
		})
	}
}

func TestEvaluate_SedFlows(t *testing.T) {
	// End-to-end: sed flows through evaluate() with the new pattern.
	t.Run("plain sed allowed", func(t *testing.T) {
		r := evaluateAll("sed 's/foo/bar/' file")
		require.NotNil(t, r)
		assert.Equal(t, "sed", r.reason)
		assert.Equal(t, decisionAllow, r.decision)
	})

	t.Run("sed -i drops to ask", func(t *testing.T) {
		r := evaluateAll("sed -i 's/foo/bar/' file")
		require.NotNil(t, r)
		assert.Equal(t, "sed", r.reason)
		assert.Equal(t, decisionAsk, r.decision)
	})
}
