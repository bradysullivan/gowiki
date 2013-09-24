package main

import (
    "html/template"
    "regexp"
    "net/http"
    "fmt"
    "strings"
    "labix.org/v2/mgo"
    "labix.org/v2/mgo/bson"
)

const lenPath = len("/view/")

var templates = template.Must(template.ParseFiles("templates/index.html", "templates/edit.html",
                                                  "templates/view.html", "templates/notfound.html",
                                                  "templates/header.html", "templates/footer.html"))
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")
var db *mgo.Collection

type Page struct {
    Title string
    Perma string
    Body []byte
}

func (p *Page) save() error {
    fmt.Println("Updating existing page.")
    _, err := db.Upsert(bson.M{"perma":p.Perma}, p)
    return err
}

func loadPage(title string) (*Page, error) {
    search := strings.ToLower(title)
    result := Page{}
    err := db.Find(bson.M{"perma":search}).One(&result)
    return &result, err
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    tmpl = tmpl + ".html"
    err := templates.ExecuteTemplate(w, tmpl, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func getPageList() ([]string, error) {
    pages := []Page{}
    query := db.Find(nil).Limit(100).Iter()
    err := query.All(&pages)
    if err != nil {
        return nil, err
    }

    titles := make([]string, len(pages))
    for i, p := range pages {
        titles[i] = p.Title
    }
    return titles, nil
}

func log(title string, action string, r *http.Request) {
    if r.Header["X-Real-Ip"] != nil {
        fmt.Printf("%s: %s by %s\n", title, action, r.Header["X-Real-Ip"][0])
    } else {
        fmt.Printf("%s: %s by %s\n", title, action, r.RemoteAddr)
    }
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
    log("Index", "viewed", r)
    if len(r.URL.Path) == 1 {
        // asked for root dir.
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
    } else { // Didn't ask for root, but we'll give it to them anyways.
        http.Redirect(w, r, "/", http.StatusFound)
    }
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
    log("View", title, r)
    p, err := loadPage(title)
    if err != nil {
        fmt.Println(err.Error())
        // Page not found.
        renderTemplate(w, "notfound", &Page{Title:title})
    } else {
        // Page found. Attempt to display.
        renderTemplate(w, "view", p)
    }
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    log("Edit", title, r)
    p, err := loadPage(title)   // Try and load the page if it exists.
    if err != nil {
        p = &Page{Title: title} // If it doesn't, create a new Page with the given title.
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    log("Save", title, r)
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body), Perma: strings.ToLower(title)}
    err := p.save()
    if err != nil {
        fmt.Println(err.Error())
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

func includeHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Path[1:]
    http.ServeFile(w, r, filename)
}

func main() {
    session, err := mgo.Dial("localhost")
    if err != nil {
        panic(err)
    }
    defer session.Close()
    session.SetSafe(&mgo.Safe{})
    db = session.DB("gowiki").C("pages")
    //db.DropCollection() // Used for debug reasons.
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))
    http.HandleFunc("/js/", includeHandler)
    http.HandleFunc("/css/", includeHandler)
    http.ListenAndServe(":54545", nil)
}
