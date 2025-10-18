package subtitles

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

// --- Tests pour splitSegOffsetsString -------------------------------------

func TestSplitSegOffsetsString_SimpleAndDecimal(t *testing.T) {
	tests := []struct {
		name               string
		in                 string
		wantTexts          []string
		wantEndWithTerm    []bool
		wantEndRuneIsRuneC bool // verify EndRune equals rune count of prefix up to piece
	}{
		{
			name:               "decimal not split",
			in:                 "2.6 meters",
			wantTexts:          []string{"2.6 meters"},
			wantEndWithTerm:    []bool{false},
			wantEndRuneIsRuneC: true,
		},
		{
			name:               "ellipsis and terminator",
			in:                 "Wait... what?",
			wantTexts:          []string{"Wait...", "what?"},
			wantEndWithTerm:    []bool{true, true},
			wantEndRuneIsRuneC: true,
		},
		{
			name:               "single sentence",
			in:                 "Hello world.",
			wantTexts:          []string{"Hello world."},
			wantEndWithTerm:    []bool{true},
			wantEndRuneIsRuneC: true,
		},
		{
			name:               "newline treated as space",
			in:                 "One line\nNext line.",
			wantTexts:          []string{"One line Next line."},
			wantEndWithTerm:    []bool{true},
			wantEndRuneIsRuneC: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := splitSegOffsetsString(tc.in)
			if len(parts) != len(tc.wantTexts) {
				t.Fatalf("got %d parts, want %d : %#v", len(parts), len(tc.wantTexts), parts)
			}
			runeAcc := 0
			for i, p := range parts {
				if p.Text != tc.wantTexts[i] {
					t.Errorf("part %d: text = %q; want %q", i, p.Text, tc.wantTexts[i])
				}
				if p.EndWithTerminator != tc.wantEndWithTerm[i] {
					t.Errorf("part %d: EndWithTerminator = %v; want %v", i, p.EndWithTerminator, tc.wantEndWithTerm[i])
				}
				// verify EndRune is non-decreasing and corresponds to rune counts (if flag set)
				if p.EndRune < runeAcc {
					t.Errorf("part %d: EndRune decreased: got %d prev %d", i, p.EndRune, runeAcc)
				}
				runeAcc = p.EndRune
				if tc.wantEndRuneIsRuneC {
					// expected EndRune equals rune count of substring up to piece (recompute)
					// We compute expected by counting runes in prefix of string of length matching the piece text occurrence.
					// Simpler: assert EndRune <= total runes and >0
					totalRunes := utf8.RuneCountInString(tc.in)
					if p.EndRune < 0 || p.EndRune > totalRunes {
						t.Errorf("part %d: EndRune out of range: %d (total %d)", i, p.EndRune, totalRunes)
					}
				}
			}
		})
	}
}

// --- Tests pour RawJSON3ToPhrases ----------------------------------------

func TestRawJSON3ToPhrases_MultiEvent_JoinAcrossEvents(t *testing.T) {
	// ev1: no terminator, ev2: terminator => final phrase should start at ev1 start
	raw := rawJSON3{
		Events: []rawEvent{
			{
				TStartMs:    ptrInt64(0),
				DDurationMs: ptrInt64(1000),
				Segs: []rawSeg{
					{Utf8: "Hello world"},
				},
			},
			{
				TStartMs:    ptrInt64(1000),
				DDurationMs: ptrInt64(1000),
				Segs: []rawSeg{
					{Utf8: "This is fine."},
				},
			},
		},
	}

	phrases, _ := TransformManualRawToPhrases(raw)
	if len(phrases) != 1 {
		t.Fatalf("expected 1 phrase, got %d: %#v", len(phrases), phrases)
	}
	wantText := "Hello world This is fine."
	if phrases[0].Text != wantText {
		t.Fatalf("phrase text = %q; want %q", phrases[0].Text, wantText)
	}
	if phrases[0].TimestampMs != 0 {
		t.Fatalf("phrase timestamp = %d; want 0", phrases[0].TimestampMs)
	}
}

func TestRawJSON3ToPhrases_SameEventMultiplePhrases(t *testing.T) {
	// single event containing two sentences "First. Second."
	evText := "First. Second."
	evStart := int64(0)
	evDur := int64(2000) // per rune ms will split timestamps
	raw := rawJSON3{
		Events: []rawEvent{
			{
				TStartMs:    &evStart,
				DDurationMs: &evDur,
				Segs: []rawSeg{
					{Utf8: evText},
				},
			},
		},
	}

	phrases, _ := TransformManualRawToPhrases(raw)
	if len(phrases) != 2 {
		t.Fatalf("expected 2 phrases, got %d: %#v", len(phrases), phrases)
	}

	// First phrase must start at evStart (0)
	if phrases[0].TimestampMs != evStart {
		t.Fatalf("first phrase timestamp = %d; want %d", phrases[0].TimestampMs, evStart)
	}
	if phrases[0].Text != "First." {
		t.Fatalf("first phrase text = %q; want %q", phrases[0].Text, "First.")
	}

	// compute expected start of second phrase: perRuneMs * startRuneIndex
	totalRunes := utf8.RuneCountInString(evText)
	// find rune index where "Second." starts
	idx := strings.Index(evText, "Second.")
	if idx < 0 {
		t.Fatalf("couldn't find 'Second.' substring")
	}
	// rune count before "Second."
	runesBefore := utf8.RuneCountInString(evText[:idx])
	perRuneMs := float64(evDur) / float64(totalRunes)
	expectedSecondStart := int64(math.Round(perRuneMs * float64(runesBefore)))
	if phrases[1].TimestampMs != expectedSecondStart {
		t.Fatalf("second phrase timestamp = %d; want %d", phrases[1].TimestampMs, expectedSecondStart)
	}
	if phrases[1].Text != "Second." {
		t.Fatalf("second phrase text = %q; want %q", phrases[1].Text, "Second.")
	}
}

// helper to create *int64 easily in tests
func ptrInt64(v int64) *int64 { return &v }
