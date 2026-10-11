// Package sdmxml — sdm.xml, внешние <dao-class>-файлы и settings.xml.
//
// Схемы (sdm.xsd, dao.xsd, settings.xsd) здесь не проверяются: XML пишут в IDE, где
// подсказки и подсветку даёт XSD. Здесь — то, без чего вывод был бы неправильным:
// умолчания атрибутов из XSD, порядок DAO-элементов (от него зависит порядок методов)
// и незнакомые элементы DAO как ошибка, а не молчаливый пропуск.
package sdmxml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Sdm struct {
	DtoClasses []DtoClass `xml:"dto-class"`
	DaoClasses []DaoClass `xml:"dao-class"`
}

type DtoClass struct {
	Name   string  `xml:"name,attr"`
	Ref    string  `xml:"ref,attr"`
	PK     string  `xml:"pk,attr"`   // умолчание "*"
	Auto   string  `xml:"auto,attr"` // умолчание "*"
	Header *string `xml:"header"`    // nil — элемента нет (в шаблоне это разные случаи)
	Custom string  `xml:"custom"`
	Fields []Field `xml:"field"`
}

type Field struct {
	Column string `xml:"column,attr"`
	Type   string `xml:"type,attr"`
}

func (c *DtoClass) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain DtoClass
	p := plain{PK: "*", Auto: "*"}
	if err := d.DecodeElement(&p, &start); err != nil {
		return err
	}
	*c = DtoClass(p)
	return nil
}

// DaoClass — <dao-class> в sdm.xml: либо ссылка на внешний файл (Ref), либо сами элементы.
type DaoClass struct {
	Name  string
	Ref   string
	Items []DaoItem // в порядке документа
}

func (c *DaoClass) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var p struct {
		Name  string    `xml:"name,attr"`
		Ref   string    `xml:"ref,attr"`
		Items []DaoItem `xml:",any"`
	}
	if err := d.DecodeElement(&p, &start); err != nil {
		return err
	}
	if err := checkItems(p.Items); err != nil {
		return fmt.Errorf("<dao-class name=\"%s\">: %w", p.Name, err)
	}
	*c = DaoClass{Name: p.Name, Ref: p.Ref, Items: p.Items}
	return nil
}

// DaoItem — один из crud, query, query-list, query-dto, query-dto-list, exec-dml.
type DaoItem struct {
	XMLName        xml.Name
	Method         string      `xml:"method,attr"`
	Ref            string      `xml:"ref,attr"`
	Dto            string      `xml:"dto,attr"`
	ExternalSQL    bool        `xml:"external-sql,attr"`
	ReturnType     string      `xml:"return-type,attr"`
	Table          string      `xml:"table,attr"`           // crud, умолчание "*"
	FetchGenerated bool        `xml:"fetch-generated,attr"` // crud, умолчание true
	Create         *CrudMethod `xml:"create"`
	ReadAll        *CrudMethod `xml:"read-all"`
	Read           *CrudMethod `xml:"read"`
	Update         *CrudMethod `xml:"update"`
	Delete         *CrudMethod `xml:"delete"`
}

// Kind — имя элемента: "crud", "query", ...
func (it *DaoItem) Kind() string {
	return it.XMLName.Local
}

type CrudMethod struct {
	Method string `xml:"method,attr"`
}

func (it *DaoItem) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain DaoItem
	p := plain{Table: "*", FetchGenerated: true}
	if err := d.DecodeElement(&p, &start); err != nil {
		return err
	}
	*it = DaoItem(p)
	return nil
}

var daoKinds = map[string]bool{
	"crud": true, "query": true, "query-list": true, "query-dto": true, "query-dto-list": true, "exec-dml": true,
}

func checkItems(items []DaoItem) error {
	for _, it := range items {
		if !daoKinds[it.Kind()] {
			return fmt.Errorf("unexpected element <%s>", it.Kind())
		}
	}
	return nil
}

// externalDao — корень внешнего DAO-файла.
type externalDao struct {
	Items []DaoItem `xml:",any"`
}

// LoadSdm читает sdmDir/sdm.xml и подтягивает элементы внешних <dao-class ref="X.xml">
// (путь — относительно sdmDir, как в Java-плагине).
func LoadSdm(sdmDir string) (*Sdm, error) {
	var s Sdm
	if err := readXML(filepath.Join(sdmDir, "sdm.xml"), "sdm", &s); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, c := range s.DtoClasses {
		if err := unique(names, c.Name); err != nil {
			return nil, err
		}
	}
	for i := range s.DaoClasses {
		c := &s.DaoClasses[i]
		if err := unique(names, c.Name); err != nil {
			return nil, err
		}
		if c.Ref == "" {
			continue
		}
		if len(c.Items) > 0 {
			return nil, fmt.Errorf("<dao-class name=\"%s\">: both ref=\"%s\" and inner elements", c.Name, c.Ref)
		}
		var ext externalDao
		path := filepath.Join(sdmDir, c.Ref)
		if err := readXML(path, "dao-class", &ext); err != nil {
			return nil, err
		}
		if err := checkItems(ext.Items); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		c.Items = ext.Items
	}
	return &s, nil
}

func unique(names map[string]bool, name string) error {
	if name == "" {
		return errors.New("empty 'name' in <dto-class> or <dao-class>")
	}
	if names[name] {
		return fmt.Errorf("duplicate name=\"%s\" in sdm.xml", name)
	}
	names[name] = true
	return nil
}

func readXML(path, root string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var probe struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if probe.XMLName.Local != root {
		return fmt.Errorf("%s: <%s> expected, got <%s>", path, root, probe.XMLName.Local)
	}
	if err := xml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
