package xpointer

import (
	"fmt"
	"strings"
)

type xpointerStackItem struct {
	seen  map[string]int
	name  string
	count int
}

type XPointerBuilder struct {
	stack  []*xpointerStackItem
	length int
}

func (x *XPointerBuilder) Push(name string) {
	x.length++
	if len(x.stack) < x.length {
		x.stack = append(x.stack, &xpointerStackItem{seen: map[string]int{name: 1}, name: name, count: 1})
		return
	}
	item := x.stack[x.length-1]
	item.name = name
	if count, ok := item.seen[name]; ok {
		item.count = count + 1
		item.seen[name] = item.count
	} else {
		item.count = 1
		item.seen[name] = 1
	}
}

func (x *XPointerBuilder) Pop() {
	x.length--
}

func (x *XPointerBuilder) String() string {
	sb := strings.Builder{}
	for i := 0; i < x.length; i++ {
		item := x.stack[i]
		sb.WriteString("/")
		sb.WriteString(item.name)
		if item.count > 1 {
			sb.WriteString("[")
			sb.WriteString(fmt.Sprint(item.count))
			sb.WriteString("]")
		}
	}
	return sb.String()
}
