package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

type Film struct {
	Title    string
	Director string
}

var films = []Film{
	{Title: "Tetangga masa gitu", Director: "Paundra"},
	{Title: "Menuju ke mana?", Director: "Pnya apa"},
	{Title: "Kesana kemari", Director: "Amil"},
}

var tmpl = template.Must(template.ParseFiles("index.html"))

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string][]Film{
		"Films": films,
	}
	tmpl.Execute(w, data)
}

func addFilmHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		time.Sleep(1 * time.Second)
		title := r.PostFormValue("title")
		director := r.PostFormValue("director")
		newFilm := Film{Title: title, Director: director}
		films = append(films, newFilm)
		tmpl.ExecuteTemplate(w, "film-list-element", newFilm)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", homePageHandler).Methods("GET")
	r.HandleFunc("/add-film/", addFilmHandler).Methods("POST")

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
