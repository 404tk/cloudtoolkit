// Package table produces a string that represents slice of structs data in a text table
package table

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// Output formats slice of structs data and writes to standard output.
func Output(slice interface{}) {
	coln, rows, err := parse(slice)
	if err != nil {
		log.Println("[-]", err)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(coln)

	for _, v := range rows {
		table.Append(v)
	}
	table.Render()
}

func FileOutput(filename string, slice interface{}) {
	coln, rows, err := parse(slice)
	if err != nil {
		log.Println("[-]", err)
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println("[-]", err)
	}
	defer file.Close()
	table := tablewriter.NewWriter(file)
	table.SetHeader(coln)

	for _, v := range rows {
		table.Append(v)
	}
	table.Render()
}

func parse(slice interface{}) (
	coln []string, // name of columns
	rows [][]string, // rows of content
	err error,
) {

	s, err := sliceconv(slice)
	if err != nil {
		return
	}
	for i, u := range s {
		v := reflect.ValueOf(u)
		t := reflect.TypeOf(u)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			t = t.Elem()
		}
		if v.Kind() != reflect.Struct {
			err = errors.New("warning: table: items of slice should be on struct value")
			return
		}
		var row []string
		for n := 0; n < v.NumField(); n++ {
			if t.Field(n).PkgPath != "" {
				continue
			}
			cn := t.Field(n).Name
			ct := t.Field(n).Tag.Get("table")
			if ct == "" {
				ct = cn
			} else if ct == "-" {
				continue
			}
			cv := fmt.Sprintf("%+v", v.FieldByName(cn).Interface())
			if len(cv) > 40 {
				cv = stringWrap(cv, 40)
			}

			if i == 0 {
				coln = append(coln, ct)
			}

			row = append(row, cv)
		}
		rows = append(rows, row)
	}
	return coln, rows, nil
}

func sliceconv(slice interface{}) ([]interface{}, error) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return nil, errors.New("warning: sliceconv: param \"slice\" should be on slice value")
	}

	l := v.Len()
	r := make([]interface{}, l)
	for i := 0; i < l; i++ {
		r[i] = v.Index(i).Interface()
	}
	return r, nil
}

func stringWrap(s string, limit int) string {
	strSlice := strings.Split(s, "")
	var result string = ""

	for len(strSlice) > 0 {
		if len(strSlice) >= limit {
			result = result + strings.Join(strSlice[:limit], "") + "\n"
			strSlice = strSlice[limit:]
		} else {
			length := len(strSlice)
			result = result + strings.Join(strSlice[:length], "")
			strSlice = []string{}
		}
	}

	return result
}
