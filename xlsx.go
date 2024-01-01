package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
)

// Build an exported Excel file
func excel(name string, products []Product) (message Message, filename string) {

	fmt.Println("Creating Excel File...")
	f := excelize.NewFile()
	var cell string
	var row string

	//set headers
	f.SetCellValue("Sheet1", "A1", "SKU")
	f.SetCellValue("Sheet1", "B1", "Description")
	f.SetCellValue("Sheet1", "C1", "Manufacturer")
	f.SetCellValue("Sheet1", "D1", "ManufacturerPart")
	f.SetCellValue("Sheet1", "E1", "ProcessRequest")
	f.SetCellValue("Sheet1", "F1", "SortingRequest")
	f.SetCellValue("Sheet1", "G1", "Unit")
	f.SetCellValue("Sheet1", "H1", "UnitPrice")
	f.SetCellValue("Sheet1", "I1", "Currency")
	f.SetCellValue("Sheet1", "J1", "Qty")

	for i := 0; i < len(products); i++ {
		row = strconv.Itoa(i + 2)
		cell = "A" + row
		f.SetCellValue("Sheet1", cell, products[i].SKU)
		cell = "B" + row
		f.SetCellValue("Sheet1", cell, *products[i].Description)
		cell = "C" + row
		f.SetCellValue("Sheet1", cell, products[i].Manufacturer)
		cell = "D" + row
		f.SetCellValue("Sheet1", cell, *products[i].ManufacturerPart)
		cell = "E" + row
		f.SetCellValue("Sheet1", cell, *products[i].ProcessRequest)
		cell = "F" + row
		f.SetCellValue("Sheet1", cell, *products[i].SortingRequest)
		cell = "G" + row
		f.SetCellValue("Sheet1", cell, *products[i].Unit)
		cell = "H" + row
		f.SetCellValue("Sheet1", cell, *products[i].UnitPrice)
		cell = "I" + row
		f.SetCellValue("Sheet1", cell, products[i].Currency)
		cell = "J" + row
		f.SetCellValue("Sheet1", cell, *products[i].OrderQty)
	}
	// Create a new sheet.
	//index := f.NewSheet("Sheet2")
	// Set value of a cell.
	// f.SetCellValue("Sheet2", "A2", "Hello world.")
	// f.SetCellValue("Sheet1", "A2", 100)
	// Set active sheet of the workbook.
	// f.SetActiveSheet(index)
	// Save spreadsheet by the given path.
	fmt.Println("Saving Excel File...")
	filename = "./orders/" + name + ".xlsx"
	if err := f.SaveAs(filename); err != nil {
		handleerror(err)
		return message, filename
	}
	return message, filename
}

func importfile(file string) {
	f, err := excelize.OpenFile(file)
	handleerror(err)
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			handleerror(err)
		}
	}()
	// Get value from cell by given worksheet name and cell reference.
	cell, err := f.GetCellValue("Sheet1", "B2")
	if err != nil {
		handleerror(err)
		return
	}
	fmt.Println(cell)
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		handleerror(err)
		return
	}
	for _, row := range rows {
		for _, colCell := range row {
			log.Debug(colCell, "\t")
		}
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// The argument to FormFile must match the name attribute
	// of the file input on the frontend
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		handleerror(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new file in the uploads directory
	dst, err := os.Create(fmt.Sprintf("./uploads/%d%s", time.Now().UnixNano(), filepath.Ext(fileHeader.Filename)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	importfile(dst.Name())

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	// at the specified destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug(w, "Upload successful")
}
