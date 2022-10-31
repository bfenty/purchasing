package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

// Build an exported Excel file
func excel(products []Product) (message Message) {

	fmt.Println("Creating Excel File...")
	f := excelize.NewFile()
	var cell string
	var row string
	for i := 0; i < len(products); i++ {
		row = strconv.Itoa(i + 1)
		cell = "A" + row
		fmt.Println(cell, ": ", products[i].SKU)
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
		f.SetCellValue("Sheet1", cell, *products[i].Qty)
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
	if err := f.SaveAs("Book1.xlsx"); err != nil {
		fmt.Println(err)
		handleerror(err)
		return message
	}
	return message
}

func importfile(file string) {
	f, err := excelize.OpenFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// Get value from cell by given worksheet name and cell reference.
	cell, err := f.GetCellValue("Sheet1", "B2")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(cell)
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, row := range rows {
		for _, colCell := range row {
			fmt.Print(colCell, "\t")
		}
		fmt.Println()
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// The argument to FormFile must match the name attribute
	// of the file input on the frontend
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
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

	fmt.Fprintf(w, "Upload successful")
}
