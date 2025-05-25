package xpointer

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type XPointer interface {
	Fragment() int
	CountCharsToPosition(dec *xml.Decoder) (uint64, error)
	String() string
}

type XPath struct {
	Names      []string
	Counts     []uint
	TextOffset uint
}

func (x *XPath) Clone() *XPath {
	names := make([]string, len(x.Names))
	counts := make([]uint, len(x.Counts))
	copy(names, x.Names)
	copy(counts, x.Counts)
	return &XPath{
		Names:      names,
		Counts:     x.Counts,
		TextOffset: x.TextOffset,
	}
}

// String implements XPointer.
func (x *XPath) String() string {
	var sb strings.Builder
	for i := range len(x.Counts) {
		sb.WriteRune('/')
		sb.WriteString(x.Names[i])
		count := x.Counts[i]
		if count > 1 {
			sb.WriteRune('[')
			fmt.Fprint(&sb, count)
			sb.WriteRune(']')
		}
	}

	fmt.Fprint(&sb, ".")
	fmt.Fprint(&sb, x.TextOffset)

	return sb.String()
}

// Fragment implements XPointer.
func (x *XPath) Fragment() int {
	return int(x.Counts[1])
}

// CountCharsToPosition implements XPointer.
func (x *XPath) CountCharsToPosition(dec *xml.Decoder) (uint64, error) {
	counter := uint64(0)
	ptr := x.Clone()
	depth := 0
	skip := 0
	for {
		token, _ := dec.Token()
		if token == nil {
			return 0, fmt.Errorf("unexpected EOF")
		}
		switch elem := token.(type) {
		case xml.EndElement:
			skip--
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
			//Skip
			break
		case xml.StartElement:
			if skip > 0 {
				// If skipping, just go deeper
				skip++
				continue
			}
			if elem.Name.Local == ptr.Names[depth] {
				ptr.Counts[depth]--
				if ptr.Counts[depth] == 0 {
					// We actually found an elemnt, go deeper
					depth++
					if depth == len(ptr.Counts) {
						// Found the elemnt
						return counter + uint64(ptr.TextOffset), nil
					}
					continue
				}
			}
			// This is not part of the path, skip
			skip = 1
		case xml.CharData:
			runeCount := utf8.RuneCount(elem)
			counter += uint64(runeCount)
		}
	}
}

var _ XPointer = &XPath{}

type DocFragmentID struct {
	DocFragment int
	ID          string
}

// String implements XPointer.
func (d *DocFragmentID) String() string {
	return fmt.Sprintf("#_doc_fragment_%d_ %s", d.DocFragment, d.ID)
}

// CountCharsToPosition implements XPointer.
func (d *DocFragmentID) CountCharsToPosition(dec *xml.Decoder) (uint64, error) {
	counter := uint64(0)
	for {
		token, _ := dec.Token()
		if token == nil {
			return 0, fmt.Errorf("unexpected EOF")
		}
		switch elem := token.(type) {
		case xml.EndElement:
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
			//Skip
			break
		case xml.StartElement:
			for _, attr := range elem.Attr {
				if !strings.EqualFold(attr.Name.Local, "id") {
					continue
				}
				if attr.Value == d.ID {
					return counter, nil
				}
			}
		case xml.CharData:
			runeCount := utf8.RuneCount(elem)
			counter += uint64(runeCount)
		}
	}
}

// Fragment implements XPointer.
func (d *DocFragmentID) Fragment() int {
	return d.DocFragment
}

var _ XPointer = &DocFragmentID{}

var docFragmentIDRegex = regexp.MustCompile(`^#_doc_fragment_(\d+)_\s+([^\s]+)\s*$`)

func Parse(s string) (XPointer, error) {
	if strings.HasPrefix(s, "#") {
		return ParseDocFragmentID(s)
	}

	return ParseXPath(s)
}

func ParseDocFragmentID(s string) (*DocFragmentID, error) {
	m := docFragmentIDRegex.FindStringSubmatch(s)
	if len(m) == 0 {
		return nil, fmt.Errorf("invalid DocFragmentID: %s", s)
	}

	frag, err := strconv.ParseUint(m[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid DocFragmentID: invalid fragment: %s", s)
	}

	return &DocFragmentID{
		DocFragment: int(frag),
		ID:          m[2],
	}, nil
}

var partRegex = regexp.MustCompile(`^(\w+|text\(\))(?:\[(\d+)\])?(?:\.(\d+))?$`)

func ParseXPath(s string) (*XPath, error) {
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return &XPath{}, nil
	}

	names := make([]string, 0, len(parts))
	counts := make([]uint, 0, len(parts))
	textOffset := uint(0)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		m := partRegex.FindStringSubmatch(part)
		if len(m) == 0 {
			return nil, fmt.Errorf("invalid XPointer path segment: invalid part %s: %s ", s, part)
		}
		names = append(names, m[1])
		count := uint(1)
		if len(m[2]) > 0 {
			c, err := strconv.ParseUint(m[2], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid XPointer `%s`: %s", s, err)
			}
			count = uint(c)
		}
		counts = append(counts, count)
		rawOffset := m[3]
		isLast := i == len(parts)-1
		if rawOffset != "" {
			if !isLast {
				return nil, fmt.Errorf("invalid XPointer path segment: char offset specified in middle of path: %s: %s", s, part)
			}
			offset, err := strconv.ParseUint(rawOffset, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid XPointer path segment: invalid char offset: %s: %s", rawOffset, s)
			}
			textOffset = uint(offset)
		}
	}

	return &XPath{Names: names, Counts: counts, TextOffset: textOffset}, nil
}
