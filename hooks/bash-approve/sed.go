package main

import (
	"mvdan.cc/sh/v3/syntax"
)

// sedSpec marks -e/-f as value-taking so parseArgs consumes their argument
// (otherwise a script like `s/foo/-i/` would be scanned as a flag cluster and
// false-positive on the in-place check).
var sedSpec = flagSpec{
	short: map[byte]string{
		'e': "expression", 'f': "file", 'n': "quiet",
		'r': "regexp-extended", 'E': "regexp-extended",
		's': "separate", 'u': "unbuffered",
	},
	takesValue: map[string]bool{
		"expression": true, "file": true,
	},
}

// isSedSafe returns false (→ ask via validateFallback) when sed is
// invoked in an in-place edit form. All other invocations pass.
// args includes the command name at args[0]; flags start at args[1].
func isSedSafe(args []*syntax.Word, _ evalContext) bool {
	if len(args) < 2 {
		return true
	}
	parsed := parseArgs(args[1:], sedSpec)
	if !parsed.allLiteral {
		// Dynamic flags/scripts can't be verified — be conservative.
		return false
	}
	// `-i`, `-i.bak`, `-ni`, etc. all leave a literal `i` key in flags after
	// short-cluster expansion (parseArgs splits combined shorts).
	if _, ok := parsed.flags["i"]; ok {
		return false
	}
	// `--in-place` and `--in-place=.bak`.
	if _, ok := parsed.flags["in-place"]; ok {
		return false
	}
	return true
}
