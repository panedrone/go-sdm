package cg

import (
	"errors"
	"strings"
	"unicode"
)

// Грамматика 'method' (MethodDeclarations.java, ExecDmlParams.java).

func parseMethodParams(src string) (before, inside string, err error) {
	src = strings.TrimSpace(src)
	pos := strings.IndexByte(src, '(')
	if pos == -1 {
		return src, "", nil
	}
	if !strings.HasSuffix(src, ")") {
		return "", "", errors.New("')' expected")
	}
	return src[:pos], src[pos+1 : len(src)-1], nil
}

// ParseMethodDeclaration — "get_something(p1, p2)" -> "get_something", "p1, p2".
func ParseMethodDeclaration(text string) (name, params string, err error) {
	name, inside, err := parseMethodParams(text)
	if err != nil || inside == "" {
		return name, "", err
	}
	params, rest, err := parseMethodParams(inside)
	if err != nil {
		return "", "", err
	}
	if rest != "" {
		return "", "", errors.New("Invalid params: " + text)
	}
	return name, params, nil
}

// SplitMappedParam — "on_dto:Dto" или "on_dto~Dto" -> {"on_dto", "Dto"}; nil — обычный параметр.
func SplitMappedParam(pd string) []string {
	var parts []string
	if strings.Contains(pd, "~") {
		parts = jsplitLit(pd, "~")
	}
	if strings.Contains(pd, ":") {
		parts = jsplitLit(pd, ":")
	}
	return parts
}

// ListedItems — "a, b, [x:X, y:Y]" -> {"a", "b", "[x:X, y:Y]"} с проверкой имён.
func ListedItems(list string, allowColon bool) ([]string, error) {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil, nil
	}
	lastArr := ""
	if pos := strings.IndexByte(list, '['); pos != -1 {
		if !strings.HasSuffix(list, "]") {
			return nil, errors.New("Ending ']' expected")
		}
		lastArr = list[pos:]
		if pos == 0 {
			return []string{lastArr}, nil
		}
		list = strings.TrimSpace(list[:pos])
		list = strings.TrimSuffix(list, ",")
	}
	items := jsplitLit(list, ",")
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
		parts := jsplit(items[i], reWS)
		var name string
		switch len(parts) {
		case 1:
			name = parts[0]
		case 2:
			name = parts[1]
		default:
			return nil, errors.New("The item is null or empty: " + list)
		}
		if err := checkItem(name, allowColon); err != nil {
			return nil, err
		}
	}
	if lastArr != "" {
		items = append(items, lastArr)
	}
	return items, nil
}

func checkItem(name string, allowColon bool) error {
	if name == "" {
		return errors.New("Item name is null or empty")
	}
	r := []rune(name)
	if (!unicode.IsLetter(r[0]) || r[0] == '$') && r[0] != '_' {
		return errors.New("Invalid starting character in the name of item: " + name)
	}
	for _, ch := range r[1:] {
		if !allowColon && ch == ':' {
			return errors.New("':' is not allowed in this context")
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && !strings.ContainsRune("$_~:-", ch) {
			return errors.New("Invalid character in the name of item: " + name)
		}
	}
	return nil
}

// splitParamTypeAndName — "java.lang.Integer p_id" -> "java.lang.Integer", "p_id"; тип "" если нет.
func splitParamTypeAndName(pd string) (typ, name string, hasType bool) {
	parts := jsplit(pd, reWS)
	if len(parts) > 1 {
		name = parts[len(parts)-1]
		return strings.TrimSpace(pd[:len(pd)-1-len(name)]), name, true
	}
	return "", pd, false
}

func checkDuplicates(names []string) error {
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			return errors.New("Duplicated parameter names")
		}
		seen[n] = true
	}
	return nil
}

func checkRequiredAttr(node, method string) error {
	if strings.TrimSpace(method) == "" {
		return errors.New("<" + node + "...\n'method' is not set.")
	}
	return nil
}

// ExecDmlSlot — один параметр exec-dml (ExecDmlParamSlot.java).
type ExecDmlSlot struct {
	Kind           ExecDmlKind
	Index          int        // позиция в объявлении
	Mapping        []string   // Mapped: {"on_dto", "Dto"}
	CursorMappings [][]string // CursorArray: то же для каждого элемента
}

type ExecDmlKind int

const (
	Plain       ExecDmlKind = iota // "p_id"
	Mapped                         // "on_dto:Dto" или "on_dto~Dto" — один неявный курсор
	CursorArray                    // "[on_dto_1:Dto1, on_dto_2:Dto2]"
)

func ParseExecDmlParams(descriptors []string) ([]ExecDmlSlot, error) {
	var res []ExecDmlSlot
	for i, d := range descriptors {
		pd := strings.TrimSpace(d)
		slot := ExecDmlSlot{Index: i}
		if strings.HasPrefix(pd, "[") && strings.HasSuffix(pd, "]") {
			items, err := ListedItems(pd[1:len(pd)-1], true)
			if err != nil {
				return nil, err
			}
			slot.Kind = CursorArray
			for _, ipd := range items {
				m := SplitMappedParam(ipd)
				if m == nil {
					return nil, errors.New("Implicit cursors are specified incorrectly." +
						" Expected syntax: [on_dto_1:Dto1, on_dto_2:Dto2, ...]. Specified: " +
						"[" + strings.Join(items, ",") + "]")
				}
				slot.CursorMappings = append(slot.CursorMappings, m)
			}
		} else if m := SplitMappedParam(pd); m != nil {
			slot.Kind, slot.Mapping = Mapped, m
		}
		res = append(res, slot)
	}
	return res, nil
}
