package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type UploadedFile struct {
	Title string
	Path  string
	Date  string
}

type DateUploads struct {
	Date  string
	Files []UploadedFile
}

type PageData struct {
	PastUploads []DateUploads
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	uploadedFiles, err := getUploadedFiles()
	if err != nil {
		http.Error(w, "Error retrieving uploaded files", http.StatusInternalServerError)
		return
	}

	var sortedUploads []DateUploads
	for date, files := range uploadedFiles {
		sortedUploads = append(sortedUploads, DateUploads{
			Date:  date,
			Files: files,
		})
	}

	data := PageData{
		PastUploads: sortedUploads,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	today := time.Now().Format("2006-01-02")
	dir := filepath.Join("uploads", today)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(dir, header.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, file)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getUploadedFiles() (map[string][]UploadedFile, error) {
	uploadedFiles := make(map[string][]UploadedFile)

	err := filepath.Walk("uploads", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath := strings.TrimPrefix(path, "uploads/")
			parts := strings.SplitN(relPath, "/", 2)
			if len(parts) == 2 {
				date := parts[0]
				uploadedFiles[date] = append(uploadedFiles[date], UploadedFile{
					Title: info.Name(),
					Path:  relPath,
					Date:  date,
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	var dates []string
	for date := range uploadedFiles {
		dates = append(dates, date)
	}

	sort.Slice(dates, func(i, j int) bool {
		timeI, _ := time.Parse("2006-01-02", dates[i])
		timeJ, _ := time.Parse("2006-01-02", dates[j])
		return timeI.After(timeJ)
	})

	sortedUploadedFiles := make(map[string][]UploadedFile)
	for _, date := range dates {
		sortedUploadedFiles[date] = uploadedFiles[date]
	}

	return sortedUploadedFiles, nil
}

func main() {
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("Server running on http://localhost:3100")
	http.ListenAndServe(":3100", nil)
}
