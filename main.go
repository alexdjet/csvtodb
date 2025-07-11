package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/yeka/zip"
)

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
			archive, err := zip.OpenReader(strFilePath)
			if err != nil {
				log.Fatal(err)
			}
			defer archive.Close()

			for _, file := range archive.File {
				if file.FileInfo().IsDir() {
					continue
				}

				file.SetPassword(zipPass)

				archivedFilePath := filepath.Join(dirPath, file.Name)
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
		} else if ext == ".csv" {
			parseCsv(strFilePath)
		}

	}
}

/**
 * распарсить содержимое, подготовить запросы в БД
 *
 * @author	Unknown
 * @since	v0.0.1
 * @global
 * @return	void
 */
func parseCsv(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err) // return

	}
	defer file.Close()

	reader := csv.NewReader(file)

	// records, err := reader.ReadAll()
	// if err != nil {
	// 	log.Fatal(err) // return
	// }

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(record)
	}
}
