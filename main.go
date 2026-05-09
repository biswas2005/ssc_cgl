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

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	err := tmpl.Execute(w, mcqs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	loadMCQs()

	http.HandleFunc("/", homeHandler)
	log.Println("Server running at http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
