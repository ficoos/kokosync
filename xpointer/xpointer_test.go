package xpointer_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ficoos/moonkosync/xpointer"
)

func xpointerCMPOpts() cmp.Options {
	return cmp.Options{
		cmpopts.EquateEmpty(),
	}
}

func TestParser(t *testing.T) {
	for _, tc := range []struct {
		Input string
		Want  xpointer.XPointer
	}{
		{
			Input: "",
			Want:  &xpointer.XPath{},
		},
		{
			Input: "/",
			Want:  &xpointer.XPath{},
		},
		{
			Input: "//",
			Want:  &xpointer.XPath{},
		},
		{
			Input: "/body/div",
			Want: &xpointer.XPath{
				Names:  []string{"body", "div"},
				Counts: []uint{1, 1},
			},
		},
		{
			Input: "/body[1]/div[1]",
			Want: &xpointer.XPath{
				Names:  []string{"body", "div"},
				Counts: []uint{1, 1},
			},
		},
		{
			Input: "/body[2]/div[3]",
			Want: &xpointer.XPath{
				Names:  []string{"body", "div"},
				Counts: []uint{2, 3},
			},
		},
		{
			Input: "/body[2]/div[3]/text().32",
			Want: &xpointer.XPath{
				Names:      []string{"body", "div", "text()"},
				Counts:     []uint{2, 3, 1},
				TextOffset: 32,
			},
		},
		{
			Input: "/body/DocFragment[41].0",
			Want: &xpointer.XPath{
				Names:      []string{"body", "DocFragment"},
				Counts:     []uint{1, 41},
				TextOffset: 0,
			},
		},
		{
			Input: "/body/DocFragment[10]/body/div/p[8]/text()[1].56",
			Want: &xpointer.XPath{
				Names:      []string{"body", "DocFragment", "body", "div", "p", "text()"},
				Counts:     []uint{1, 10, 1, 1, 8, 1},
				TextOffset: 56,
			},
		},
		{
			Input: "#_doc_fragment_8_ pt1",
			Want: &xpointer.DocFragmentID{
				DocFragment: 8,
				ID:          "pt1",
			},
		},
	} {
		t.Run("Parse `"+tc.Input+"`", func(t *testing.T) {
			got, err := xpointer.Parse(tc.Input)
			if err != nil {
				t.Fatalf("parse `%s`: %s", tc.Input, err)
			}

			if diff := cmp.Diff(tc.Want, got, xpointerCMPOpts()); diff != "" {
				t.Errorf("xpointer.Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
