package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caarlos0/env"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/yeka/zip"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type Config struct {
	ZipPass string `env:"ZIP_PASS" envDefault:"pass"`
	DbHost  string `env:"DB_HOST" envDefault:"localhost"`
	DbPort  string `env:"DB_PORT" envDefault:"3306"`
	DbUser  string `env:"DB_USER" envDefault:"user"`
	DbPass  string `env:"DB_PASSWORD" envDefault:"password"`
	DbName  string `env:"DB_NAME" envDefault:"dbname"`
}

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

	// Загружаем .env файл в окружение
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Error load enviroment variables in .env file: %v", err)
	}

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", cfg.DbUser, cfg.DbPass, cfg.DbHost, cfg.DbPort, cfg.DbName)
	fmt.Println(dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
			unzip(&cfg, dirPath, strFilePath)
			fmt.Printf("Is csv %v\n", strFilePath)
			rows := parseCsv(strFilePath)
			sendToDb(db, rows)
			continue
		}

		if ext == ".csv" {
			fmt.Printf("Is csv %v\n", strFilePath)
			rows := parseCsv(strFilePath)
			sendToDb(db, rows)
		}

	}
}

func unzip(cfg *Config, destDir, file string) {

	fmt.Println("PASS: ", cfg.ZipPass)

	archive, err := zip.OpenReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}

		file.SetPassword(cfg.ZipPass)

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

func parseCsv(filePath string) []RowReport {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err) // return
	}
	defer file.Close()

	chReader := transform.NewReader(file, charmap.Windows1251.NewDecoder())

	reader := csv.NewReader(chReader)
	reader.LazyQuotes = true
	reader.Comma = ';'
	reader.FieldsPerRecord = 11

	// Skip header rows
	_, _ = reader.Read()
	_, _ = reader.Read()

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
			break
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

	return rows
	// добавить одним запросом
	// обработать ошибку на уникальные столбцы
	// err = db.Insert(rows...)
}

func sendToDb(db *sql.DB, rows []RowReport) {

	for _, row := range rows {
		_, err := insertRow(db, &row)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			// fmt.Println("Вставлена запись с ID:", id)
			fmt.Println("Вставлена новая запись.")
		}
	}
}

func insertRow(db *sql.DB, row *RowReport) (int64, error) {
	query := "INSERT INTO payments VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	result, err := db.Exec(query,
		row.Operation,
		row.Date,
		row.Amount,
		row.Fee,
		row.Currency,
		row.MerchantAccountID,
		row.UserAccountID,
		row.Status,
		row.Category,
		row.ClientTransactionID,
		row.Description)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {

			if mysqlErr.Number == 1062 {
				return 0, fmt.Errorf("duplicate entry error: %w", err)
			} else {
				return 0, fmt.Errorf("MySQL ошибка №%d: %s\n", mysqlErr.Number, mysqlErr.Message)
			}
		} else {
			return 0, fmt.Errorf("Ошибка не MySQL:", err)
		}
	}

	return result.LastInsertId()
}
