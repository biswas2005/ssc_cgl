package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type MCQ struct {
	ID          int      `json:"id"`
	Subject     string   `json:"subject"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Answer      string   `json:"answer"`
	Description string   `json:"description"`
}

var mcqs []MCQ

func loadMCQs() {
	file, err := os.ReadFile("mcqs.json")
	if err != nil {
		log.Fatal("Error reading JSON file:", err)
	}
	err = json.Unmarshal(file, &mcqs)
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
	}
}

func saveMCQs() {
	data, err := json.MarshalIndent(mcqs, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("mcqs.json", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	subjectFilter := r.URL.Query().Get("subject")
	var filteredMCQs []MCQ

	if subjectFilter == "" {
		filteredMCQs = mcqs
	} else {
		for _, mcq := range mcqs {
			if mcq.Subject == subjectFilter {
				filteredMCQs = append(filteredMCQs, mcq)
			}
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err := tmpl.Execute(w, filteredMCQs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func addPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/add.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func addMCQHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/add", http.StatusSeeOther)
		return
	}

	subject := r.FormValue("subject")
	question := r.FormValue("question")

	option1 := r.FormValue("option1")
	option2 := r.FormValue("option2")
	option3 := r.FormValue("option3")
	option4 := r.FormValue("option4")

	answer := r.FormValue("answer")
	description := r.FormValue("description")

	newMCQ := MCQ{
		ID:       len(mcqs) + 1,
		Subject:  subject,
		Question: question,
		Options: []string{
			option1,
			option2,
			option3,
			option4,
		},
		Answer:      answer,
		Description: description,
	}
	mcqs = append(mcqs, newMCQ)
	saveMCQs()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func chatGPTImportPageHandler(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(
		template.ParseFiles("templates/chatgpt_import.html"),
	)

	err := tmpl.Execute(w, nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func chatGPTSubmitHandler(w http.ResponseWriter, r *http.Request) {

	optionRegex := regexp.MustCompile(
		`^(\(?[A-Da-d1-4]\)|[A-Da-d1-4]\.|[•\-\*])\s*(.+)`,
	)

	answerMap := map[string]int{
		"A": 0,
		"B": 1,
		"C": 2,
		"D": 3,

		"a": 0,
		"b": 1,
		"c": 2,
		"d": 3,

		"1": 0,
		"2": 1,
		"3": 2,
		"4": 3,

		"(A)": 0,
		"(B)": 1,
		"(C)": 2,
		"(D)": 3,
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/chatgpt-import", http.StatusSeeOther)
		return
	}

	rawText := r.FormValue("mcqs")

	// Split by numbered questions
	re := regexp.MustCompile(`(?m)^\d+\.`)

	parts := re.Split(rawText, -1)

	for _, part := range parts {

		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		lines := strings.Split(part, "\n")

		var mcq MCQ

		mcq.ID = len(mcqs) + 1

		var options []string

		for i, line := range lines {

			line = strings.TrimSpace(line)

			// First line = Question
			if i == 0 {
				mcq.Question = line
				continue
			}

			// Detect Answer
			if strings.HasPrefix(line, "Answer:") {

				answerKey := strings.TrimSpace(
					strings.TrimPrefix(line, "Answer:"),
				)

				if idx, ok := answerMap[answerKey]; ok &&
					idx < len(options) {

					mcq.Answer = options[idx]
				}

				continue
			}

			// Detect Explanation
			if strings.HasPrefix(line, "Explanation:") {

				mcq.Description = strings.TrimSpace(
					strings.TrimPrefix(line, "Explanation:"),
				)

				continue
			}

			// Detect Options
			matches := optionRegex.FindStringSubmatch(line)

			if len(matches) > 2 {

				optionText := strings.TrimSpace(matches[2])

				options = append(options, optionText)
			}
		}

		mcq.Options = options

		// Default Subject
		mcq.Subject = "General"

		if mcq.Question != "" {
			mcqs = append(mcqs, mcq)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	loadMCQs()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addPageHandler)
	http.HandleFunc("/submit", addMCQHandler)
	http.HandleFunc("/chatgpt-import", chatGPTImportPageHandler)
	http.HandleFunc("/chatgpt-submit", chatGPTSubmitHandler)

	log.Println("Server running at http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
