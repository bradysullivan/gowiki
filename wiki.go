package main

import (
    "html/template"
    "io/ioutil"
    "regexp"
    "net/http"
    "strings"
)

const lenPath = len("/view/")

var templates = template.Must(template.ParseFiles("templates/index.html", "templates/edit.html", "templates/view.html", "templates/notfound.html"))
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

type Page struct {
    Title string
    Body []byte
}

func (p *Page) save() error {
    filename := "pages/" + p.Title + ".txt"
    return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
    filename := "pages/" + title + ".txt"
    body, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: title, Body:body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    tmpl = tmpl + ".html"
    err := templates.ExecuteTemplate(w, tmpl, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func getPageList() ([]string, error) {
        files, err := ioutil.ReadDir("pages")
        if err != nil {
            return nil, err
        }
        pages := make([]string, len(files))
        for i, p := range files {
            pages[i] = strings.Replace(p.Name(), ".txt", "", 1)
        }
        return pages, nil
}


func rootHandler(w http.ResponseWriter, r *http.Request) {
    if len(r.URL.Path) == 1 {
        // localhost/
        pages, err := getPageList()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        err = templates.ExecuteTemplate(w, "index.html", pages)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    } else {
        http.Redirect(w, r, "/", http.StatusFound)
    }
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        // Page not found.
        renderTemplate(w, "notfound", &Page{Title:title})
    } else {
        // Page found. Attempt to display.
        renderTemplate(w, "view", p)
    }
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)   // Try and load the page if it exists.
    if err != nil {
        p = &Page{Title: title} // If it doesn't, create a new Page with the given title.
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func makeHandler( fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        title := r.URL.Path[lenPath:]
        if !titleValidator.MatchString(title) {
            http.NotFound(w, r)
            return
        }
        fn(w, r, title)
    }
}

func main() {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))
    http.ListenAndServe(":54545", nil)
}
