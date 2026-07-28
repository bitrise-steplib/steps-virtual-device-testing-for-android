package maintenance

import (
	"strings"
	"testing"
	"time"
)

const description = `One device configuration per line.

Available devices and their OS versions (generated on 2020-01-01):
` + "```" + `
┌───────────┬────────┐
│ MODEL_ID  │ MAKE   │
│ Pixel2.arm│ Google │
└───────────┴────────┘
` + "```" + `

For the authoritative list, see somewhere.
`

func TestReplaceTableBlock(t *testing.T) {
	updated, err := replaceTableBlock(description, "NEW TABLE\nSECOND ROW")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(updated, "Pixel2.arm") {
		t.Error("the old table is still present")
	}
	if !strings.Contains(updated, "NEW TABLE\nSECOND ROW") {
		t.Error("the new table was not inserted verbatim")
	}
	if !strings.Contains(updated, "generated on "+time.Now().Format(time.DateOnly)) {
		t.Error("the generated on date was not refreshed")
	}
	// Everything outside the fenced block has to survive untouched.
	if !strings.HasPrefix(updated, "One device configuration per line.\n\n") {
		t.Error("the text before the table changed")
	}
	if !strings.HasSuffix(updated, "```\n\nFor the authoritative list, see somewhere.\n") {
		t.Errorf("the text after the table changed: %q", updated[len(updated)-60:])
	}
	if strings.Count(updated, "```") != 2 {
		t.Errorf("expected exactly one fenced block, got %d fences", strings.Count(updated, "```"))
	}
}

func TestReplaceTableBlockRejectsMalformedDescriptions(t *testing.T) {
	tests := map[string]string{
		"no header line":                 "Just prose.\n",
		"header without date":            "Available devices:\n```\ntable\n```\n",
		"header not followed by a fence": "Available devices (generated on 2020-01-01):\nprose\n```\ntable\n```\n",
		"unclosed fence":                 "Available devices (generated on 2020-01-01):\n```\ntable\n",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := replaceTableBlock(input, "NEW"); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

func TestBlockScalarRange(t *testing.T) {
	lines := strings.Split(strings.Join([]string{
		"opts:",               // 0
		"  description: |",    // 1
		"      first",         // 2
		"",                    // 3, blank line inside the block
		"      last",          // 4
		"",                    // 5, blank line after the block
		"  is_required: true", // 6
	}, "\n"), "\n")

	first, last, indent, err := blockScalarRange(lines, 2)
	if err != nil {
		t.Fatal(err)
	}
	if first != 2 {
		t.Errorf("first = %d, want 2", first)
	}
	// The trailing blank line belongs to what follows, not to the block.
	if last != 5 {
		t.Errorf("last = %d, want 5 (the trailing blank line must be left out)", last)
	}
	if indent != "      " {
		t.Errorf("indent = %q, want six spaces", indent)
	}
}

func TestBlockScalarRangeRejectsUnindentedContent(t *testing.T) {
	// A leading blank line inside the block used to yield an empty indent, which matches every
	// line and ran the range to the end of the file, wiping out everything after the block.
	tests := map[string][]string{
		"blank first line then unindented": {"  description: |", "", "not indented", "next: value"},
		"no content at all":                {"  description: |"},
		"only blank lines":                 {"  description: |", "", ""},
	}

	for name, lines := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := blockScalarRange(lines, 1); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

func TestBlockScalarRangeKeepsIndentedContentAfterABlankLine(t *testing.T) {
	// A blank line before the first content line is legal, and the indent must come from the
	// first non-blank line rather than from the blank one.
	lines := []string{"  description: |", "", "      text", "  next: value"}

	first, last, indent, err := blockScalarRange(lines, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || last != 3 || indent != "      " {
		t.Errorf("got first=%d last=%d indent=%q, want 1, 3, six spaces", first, last, indent)
	}
}
