package sdmxml

import (
	"fmt"
	"path/filepath"
)

type Settings struct {
	Folders Folders `xml:"folders"`
	Dto     Dto     `xml:"dto"`
	Dao     Dao     `xml:"dao"`
	Jdbc    Jdbc    `xml:"jdbc"`
	TypeMap TypeMap `xml:"type-map"`
	Macros  []Macro `xml:"macros>macro"`
	// <ide> — настройка IDE, CLI её не читает
}

type Folders struct {
	Target string `xml:"target,attr"` // для Go — module из go.mod
	SQL    string `xml:"sql,attr"`
}

type Dto struct {
	Scope          string `xml:"scope,attr"`
	Inheritance    string `xml:"inheritance,attr"`
	FieldNamesMode int    `xml:"field-names-mode,attr"`
	Macro          string `xml:"macro,attr"`
}

type Dao struct {
	Scope           string `xml:"scope,attr"`
	FieldAssignMode int    `xml:"field-assign-mode,attr"`
	Macro           string `xml:"macro,attr"`
}

// Jdbc — jar и class не нужны: драйвер вшит в бинарник.
type Jdbc struct {
	URL  string `xml:"url,attr"`
	User string `xml:"user,attr"`
	Pwd  string `xml:"pwd,attr"`
}

type TypeMap struct {
	Default string `xml:"default,attr"`
	Types   []Type `xml:"type"`
}

type Type struct {
	Detected string `xml:"detected,attr"`
	Target   string `xml:"target,attr"`
}

// Macro — либо value (подстановка текста), либо Velocity-код в <vm>/<vm-xml>.
type Macro struct {
	Name  string  `xml:"name,attr"`
	Value *string `xml:"value,attr"` // nil — атрибута нет
	VM    *string `xml:"vm"`
	VMXML *struct {
		Inner string `xml:",innerxml"`
	} `xml:"vm-xml"`
}

// IsVelocity — макрос на Velocity, которого в Go-версии нет.
func (m *Macro) IsVelocity() bool {
	return m.VM != nil || m.VMXML != nil
}

func LoadSettings(sdmDir string) (*Settings, error) {
	var s Settings
	path := filepath.Join(sdmDir, "settings.xml")
	if err := readXML(path, "settings", &s); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, t := range s.TypeMap.Types {
		if seen[t.Detected] {
			return nil, fmt.Errorf("%s: duplicate <type detected=\"%s\"", path, t.Detected)
		}
		seen[t.Detected] = true
	}
	macros := map[string]bool{}
	for _, m := range s.Macros {
		if macros[m.Name] {
			return nil, fmt.Errorf("%s: duplicate <macro name=\"%s\"", path, m.Name)
		}
		macros[m.Name] = true
	}
	return &s, nil
}
