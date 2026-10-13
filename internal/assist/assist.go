// Package assist makes the starting XML from the database schema, as the XML assistants of the
// Java plug-in do (SdmUtils.get_crud_sdm_xml, create_crud_xml_jaxb_dao_class, add_fk_access),
// with the defaults of their dialog: no schema, views included, English plural to singular for
// DTO class names, FK access, classes already in use skipped; <crud> is the short one, without
// inner elements (the "crud-auto" choice of the dialog: default method names). Method and
// parameter names are those of a Go project (lower camel case).
package assist

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/panedrone/go-sdm/internal/cg"
	"github.com/panedrone/go-sdm/internal/meta"
)

// DtoClasses returns a <dto-class> for each table and view, except the classes of inUse.
func DtoClasses(cat meta.Catalog, inUse map[string]bool) (string, error) {
	tables, err := cat.Tables(true)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range tables {
		name := DtoClassName(t, true)
		if !inUse[name] {
			fmt.Fprintf(&b, "<dto-class name=\"%s\" ref=\"%s\"/>\n", attr(name), attr(t))
		}
	}
	return b.String(), nil
}

// DaoItems returns, all in one pile, a <crud> for each table and view whose DTO class is not
// in inUse (DTO classes of the existing <crud>), each followed by <query-dto-list> FK access.
func DaoItems(cat meta.Catalog, inUse map[string]bool) (string, error) {
	tables, err := cat.Tables(true)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range tables {
		dto := DtoClassName(t, true)
		if inUse[dto] {
			continue
		}
		fmt.Fprintf(&b, "<crud dto=\"%s\" table=\"%s\"/>\n", attr(dto), attr(t))
		keys, err := cat.ImportedKeys(t)
		if err != nil {
			continue // as in Java: the <crud> stays, the FK access of the table is skipped
		}
		fkAccess(&b, t, keys)
	}
	return b.String(), nil
}

// fkAccess: a "get<FkTable>By<PkClass>" for each referenced table, all FK columns to it as
// the parameters and the shortcut "fk_table(col, ...)" as the SQL.
func fkAccess(b *strings.Builder, fkTable string, keys []meta.ImportedKey) {
	var pkTables []string
	cols := map[string][]string{}
	for _, k := range keys {
		if _, ok := cols[k.PKTable]; !ok {
			pkTables = append(pkTables, k.PKTable)
		}
		cols[k.PKTable] = append(cols[k.PKTable], k.FKColumn)
	}
	for _, pk := range pkTables {
		params := make([]string, len(cols[pk]))
		for i, c := range cols[pk] {
			params[i] = cg.LowerCamel(c)
		}
		method := "get" + DtoClassName(fkTable, false) + "By" + DtoClassName(pk, true) +
			"(" + strings.Join(params, ", ") + ")"
		ref := fkTable + "(" + strings.Join(cols[pk], ", ") + ")"
		fmt.Fprintf(b, "<query-dto-list dto=\"%s\" method=\"%s\" ref=\"%s\"/>\n",
			attr(DtoClassName(fkTable, true)), attr(method), attr(ref))
	}
}

// DtoClassName is table_name_to_dto_class_name: the title-cased table name, with the last
// word made singular if singular.
func DtoClassName(table string, singular bool) string {
	word := cg.Title(table)
	if !singular {
		return word
	}
	r := []rune(word)
	last := -1
	for i := len(r) - 1; i >= 0; i-- {
		if unicode.IsUpper(r[i]) {
			last = i
			break
		}
	}
	if last < 0 {
		return word // Java throws here; the name is left as is
	}
	w := []rune(singularOf(string(r[last:])))
	if len(w) == 0 {
		return word
	}
	return string(r[:last]) + string(unicode.ToUpper(w[0])) + strings.ToLower(string(w[1:]))
}

var attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func attr(s string) string {
	return attrEscaper.Replace(s)
}
