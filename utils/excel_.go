package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"

	excelize "github.com/xuri/excelize/v2"
)

const (
	defaultSheetName = "Sheet1"
	defaultTagName   = "xlsx"
	defaultSep       = "-"
)

func WriteXlsx(xlsx *excelize.File, sheetName string, records []interface{}) (*excelize.File, error) {
	return writeToBuffer(xlsx, sheetName, records)
}

func writeToBuffer(xlsx *excelize.File, sheetName string, records []interface{}) (*excelize.File, error) {
	var err error
	xlsx.NewSheet(sheetName)
	for i, t := range records {
		d := reflect.TypeOf(t).Elem()
		for j := 0; j < d.NumField(); j++ {
			var (
				column     string
				columnName string
			)
			columns := strings.Split(d.Field(j).Tag.Get(defaultTagName), defaultSep)
			if len(columns) == 2 {
				column = columns[0]
				columnName = columns[1]
			}

			if i == 0 {
				A1 := fmt.Sprintf("%s%d", column, i+1)
				err = xlsx.SetCellValue(sheetName, A1, columnName)
				if err != nil {
					log.Println(err)
				}
				style, err := xlsx.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{
						WrapText: true,
					},
					Font: &excelize.Font{
						Bold: true,
						Size: 12,
					},
				})
				if err != nil {
					log.Println(err)
				}

				err = xlsx.SetCellStyle(sheetName, A1, A1, style)
				if err != nil {
					log.Println(err)
				}
			}
			err = xlsx.SetCellValue(sheetName, fmt.Sprintf("%s%d", column, i+2), reflect.ValueOf(t).Elem().Field(j).Interface())
			if err != nil {
				log.Println(err)
			}
		}
	}
	return xlsx, nil
}

func readToStruct(reader io.Reader, sheetName string, result interface{}) error {
	xlsx, err := excelize.OpenReader(reader)
	if err != nil {
		return err
	}

	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		return err
	}

	if len(rows) <= 1 {
		return nil
	}

	columns := rows[0]
	columnJson := getColumnJson(result)
	var records []map[string]interface{}
	for i, row := range rows {
		if i == 0 {
			continue
		}
		record := make(map[string]interface{})

		for f, c := range row {
			column := columns[f]
			if j, ok := columnJson[column]; ok {
				record[j] = c
			}
		}
		records = append(records, record)
	}

	bs, _ := json.Marshal(records)
	err = json.Unmarshal(bs, &result)

	return err
}

func getColumnJson(model interface{}) map[string]string {
	columnJson := make(map[string]string)
	d := reflect.TypeOf(model).Elem().Elem()
	for j := 0; j < d.NumField(); j++ {
		var columnName string
		columns := strings.Split(d.Field(j).Tag.Get(defaultTagName), defaultSep)
		if len(columns) == 2 {
			columnName = columns[1]
		}
		columnJson[columnName] = d.Field(j).Name
	}
	return columnJson
}
