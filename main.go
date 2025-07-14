package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yeka/zip"
)

type RowReport struct {
	Operation           string
	Date                string
	Amount              float64
	Fee                 float64
	Currency            string
	MerchantAccountID   string
	UserAccountID       string
	Status              string
	Category            string
	ClientTransactionID string
	Description         string
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error load enviroment variables in .env file: %v", err)
	}

	// or os.lookupenv
	zipPass := os.Getenv("ZIP_PASS")
	// dbUser := os.Getenv("DB_USER")
	// dbPass := os.Getenv("DB_PASSWORD")

	dirPath := "../eattachs/data"

	// сканировать директорию, выбрать табличные файлы
	fd, err := os.Open(dirPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	names, _ := fd.Readdirnames(0)
	for _, filename := range names {
		fmt.Printf("File %s\n", filename)

		strFilePath := filepath.Join(dirPath, filename)
		file, err := os.Open(strFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			log.Fatal(err)
		}
		if fileInfo.IsDir() {
			continue
		}

		ext := filepath.Ext(strFilePath)
		// fmt.Println(ext)

		if ext == ".zip" {
			unzip(zipPass, dirPath, strFilePath)
			continue
		} else if ext == ".csv" {
			fmt.Printf("Is csv %v\n", strFilePath)
			parseCsv(strFilePath)
		}

	}
}

func unzip(password, destDir, file string) {
	archive, err := zip.OpenReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}

		file.SetPassword(password)

		archivedFilePath := filepath.Join(destDir, file.Name)
		fmt.Printf("Archived file %s\n", archivedFilePath)

		print(file.FileInfo())

		rr, err := file.Open()
		if err != nil {
			log.Printf("Error opening file %s: %v", archivedFilePath, err)
			continue
		}
		defer rr.Close()

		// create dest
		outFile, err := os.OpenFile(archivedFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			log.Printf("Error creating file %s: %v", archivedFilePath, err)
			continue
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rr)
		if err != nil {
			log.Printf("Error copying content for file %s: %v", file.Name, err)
		}

		fmt.Printf("Extracted: %s\n", archivedFilePath)

	}

	fmt.Println("Unzipping complete.")
}

func parseCsv(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err) // return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.Comma = ';'
	// reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = 11

	// Skip header rows
	// _, _ = reader.Read()
	// if err != nil && err != io.EOF {
	// 	fmt.Println("Error reading header:", err)
	// 	return
	// }

	var rows []RowReport

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if errors.Is(err, csv.ErrFieldCount) {
			continue
		}

		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		amount, _ := strconv.ParseFloat(strings.Replace(row[2], ",", ".", 1), 64)
		free, _ := strconv.ParseFloat(strings.Replace(row[3], ",", ".", 1), 64)

		rowReport := RowReport{
			Operation:           row[0],
			Date:                row[1],
			Amount:              amount,
			Fee:                 free,
			Currency:            row[4],
			MerchantAccountID:   row[5],
			UserAccountID:       row[6],
			Status:              row[7],
			Category:            row[8],
			ClientTransactionID: row[9],
			Description:         row[10],
		}

		rows = append(rows, rowReport)
	}

	// for _, r := range rows {
	// 	fmt.Print(r)
	// }

	// err = db.Insert(rows...)
}
