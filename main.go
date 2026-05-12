package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Fixed list of subjects. Acts as an allowlist to prevent file traversal or mixing.
var allowedSubjects = map[string]bool{
	"mathematics":           true,
	"english":               true,
	"general-science":       true,
	"history":               true,
	"geography":             true,
	"polity":                true,
	"economics":             true,
	"reasoning":             true,
	"computer":              true,
	"current-affairs":       true,
	"general-awareness":     true,
	"quantitative-aptitude": true,
}

type MCQ struct {
	ID           int      `json:"id"`
	Question     string   `json:"question"`
	Choices      []string `json:"choices"`
	CorrectIndex int      `json:"correct_index"`
	Explanation  string   `json:"explanation"`
}

func main() {
	// 1. Serve static assets (CSS/JS)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Homepage
	http.HandleFunc("/", indexHandler)

	// 3. Explicitly register each subject route & API endpoint
	for subject := range allowedSubjects {
		http.HandleFunc("/"+subject, subjectPageHandler)
		http.HandleFunc("/api/mcqs/"+subject, mcqAPIHandler)
	}

	http.HandleFunc("/admin", adminPageHandler)
	http.HandleFunc("/api/add-bulk-mcqs", addBulkMCQsHandler)

	fmt.Println("SSC Prep Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "templates/index.html")
}

func addBulkMCQsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", 405)
		return
	}

	var payload struct {
		Subject string `json:"subject"`
		Raw     string `json:"raw"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		writeJSONError(w, "Invalid JSON", 400)
		return
	}

	if !allowedSubjects[payload.Subject] {
		writeJSONError(w, "Invalid subject", 400)
		return
	}

	path := filepath.Join("data", "mcqs", payload.Subject+".json")

	var existing []MCQ

	fileData, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(fileData, &existing)
	}

	payload.Raw = strings.ReplaceAll(payload.Raw, "\r\n", "\n")
	payload.Raw = strings.ReplaceAll(payload.Raw, "∘", " degrees ")
	payload.Raw = strings.ReplaceAll(payload.Raw, "\t", " ")

	blocks := strings.Split(payload.Raw, "\n\n")

	for _, block := range blocks {

		block = strings.TrimSpace(block)

		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")

		// Clean lines
		var cleaned []string

		for _, line := range lines {

			line = strings.TrimSpace(line)

			if line != "" {
				cleaned = append(cleaned, line)
			}
		}

		lines = cleaned

		if len(lines) < 7 {
			fmt.Println("Skipping invalid block:", lines)
			continue
		}

		question := lines[0]

		choices := []string{
			strings.TrimSpace(strings.TrimPrefix(lines[1], "A)")),
			strings.TrimSpace(strings.TrimPrefix(lines[2], "B)")),
			strings.TrimSpace(strings.TrimPrefix(lines[3], "C)")),
			strings.TrimSpace(strings.TrimPrefix(lines[4], "D)")),
		}

		answerLine := lines[5]
		explanationLine := lines[6]

		answerLetter := strings.TrimSpace(
			strings.TrimPrefix(answerLine, "Answer:"),
		)

		correctIndex := 0

		switch answerLetter {
		case "A":
			correctIndex = 0
		case "B":
			correctIndex = 1
		case "C":
			correctIndex = 2
		case "D":
			correctIndex = 3
		}

		explanation := strings.TrimSpace(
			strings.TrimPrefix(explanationLine, "Explanation:"),
		)

		mcq := MCQ{
			ID:           len(existing) + 1,
			Question:     question,
			Choices:      choices,
			CorrectIndex: correctIndex,
			Explanation:  explanation,
		}

		existing = append(existing, mcq)
	}

	fmt.Println("Total MCQs:", len(existing))

	updated, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		writeJSONError(w, "Failed to encode JSON", 500)
		return
	}

	err = os.WriteFile(path, updated, 0644)
	if err != nil {
		writeJSONError(w, "Failed to save file", 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "MCQs added successfully",
	})
}

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func subjectPageHandler(w http.ResponseWriter, r *http.Request) {
	subject := strings.TrimPrefix(r.URL.Path, "/")

	// Serve the subject's HTML template
	templatePath := filepath.Join("templates", "subjects", subject+".html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		http.Error(w, "Subject page not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, templatePath)
}

func mcqAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Extract subject from URL: /api/mcqs/mathematics -> "mathematics"
	subject := strings.TrimPrefix(r.URL.Path, "/api/mcqs/")
	subject = strings.ToLower(strings.TrimSpace(subject))

	// Security: Validate against allowlist
	if !allowedSubjects[subject] {
		writeJSONError(w, "Invalid subject requested", http.StatusNotFound)
		return
	}

	// Map to isolated JSON file
	jsonPath := filepath.Join("data", "mcqs", subject+".json")

	// Extra safety: prevent directory traversal
	cleanPath := filepath.Clean(jsonPath)
	if !strings.HasPrefix(cleanPath, "data"+string(filepath.Separator)) {
		writeJSONError(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Read & validate JSON
	file, err := os.Open(cleanPath)
	if err != nil {
		writeJSONError(w, "MCQ data not found for this subject", http.StatusNotFound)
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, "Failed to read MCQ data", http.StatusInternalServerError)
		return
	}

	// Validate JSON structure
	var data []map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		writeJSONError(w, "Malformed JSON in MCQ file", http.StatusInternalServerError)
		return
	}

	// Serve data
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(raw)
}

// Helper to return standardized JSON errors
func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
