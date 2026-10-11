package main

import (
	"context"
	"fmt"
	"go_sqlite3_northwindEF/raw_sql/dbal"
)

func main() {
	ds := dbal.Ds()
	err := ds.Open()
	if err != nil {
		print(err)
		return
	}
	ctx := context.Background()
	dao := dbal.NewCustomersDao()
	res, err := dao.GetCustomersAndSuppliersByCity(ctx)
	if err != nil {
		print(err)
		return
	}
	for _, r := range res {
		line := fmt.Sprintf("%s\t%s\t%s\t%s", r.City, r.CompanyName, r.ContactName, r.Relationship)
		fmt.Println(line)
	}
}
