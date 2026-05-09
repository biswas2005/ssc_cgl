package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
)

type MCQ struct {
	ID          int      `json:"id"`
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
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	err := tmpl.Execute(w, mcqs)
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

	question := r.FormValue("question")

	option1 := r.FormValue("option1")
	option2 := r.FormValue("option2")
	option3 := r.FormValue("option3")
	option4 := r.FormValue("option4")

	answer := r.FormValue("answer")
	description := r.FormValue("description")

	newMCQ := MCQ{
		ID:       len(mcqs) + 1,
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

func main() {
	loadMCQs()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addPageHandler)
	http.HandleFunc("/submit", addMCQHandler)

	log.Println("Server running at http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
