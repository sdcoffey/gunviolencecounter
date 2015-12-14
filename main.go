package main

import (
	"database/sql"
	"fmt"
	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
	"strings"
	"encoding/json"
)

const base_url = "http://www.gunviolencearchive.org/mass-shooting"

type Incident struct {
	Id      string
	Date    time.Time
	City    string
	State   string
	Address string
	Killed  int
	Injured int
	Source  string
}

type Email struct {
	Email string
	State string
}

func main() {
	dbMap := initDb()
	go refreshData(dbMap)
	go refreshLoop(dbMap, 6*time.Hour)

	r := mux.NewRouter()
	r.HandleFunc("/email", func(writer http.ResponseWriter, req *http.Request) {
		addEmail(writer, req, dbMap)
	}).Methods("POST")
	r.HandleFunc("/victimCount", func (writer http.ResponseWriter, req *http.Request) {
		getCount(writer, dbMap)
	}).Methods("GET")

	println("UP")
	http.ListenAndServe(":3000", r)
}

func getCount(writer http.ResponseWriter, dbmap *gorp.DbMap) {
	if num, err := dbmap.SelectInt("select sum(killed + injured) as thesum from Incident"); err != nil {
		writer.WriteHeader(400)
	} else {
		writer.WriteHeader(200)
		writer.Write([]byte(fmt.Sprint(num)))
	}
}

func addEmail(writer http.ResponseWriter, req *http.Request, dbMap *gorp.DbMap) {
	var submission Email
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&submission); err != nil {
		writer.WriteHeader(400)
	} else if strings.Contains(submission.Email, "@") && strings.Contains(submission.Email, ".") && submission.State != "" {
		println("adding " + submission.Email + " " + submission.State)
		email := Email{Email: submission.Email, State: submission.Email}
		if key, _ := dbMap.SelectStr("select Email from Email where Email = ? && State = ?", email.Email, email.State); key == "" {
			dbMap.Insert(&email)
			writer.WriteHeader(200)
		}
	} else {
		writer.WriteHeader(400)
	}
}

func initDb() *gorp.DbMap {
	db, err := sql.Open("sqlite3", "data.db")
	if err != nil {
		panic(err)
	}
	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbMap.AddTable(Incident{}).SetKeys(false, "Id")
	dbMap.AddTable(Email{}).SetKeys(false, "Email")
	dbMap.CreateTablesIfNotExists()

	return dbMap
}

func refreshLoop(dbMap *gorp.DbMap, d time.Duration) {
	refreshTimer := time.Tick(d)
	for range refreshTimer {
		refreshData(dbMap)
	}
}

func refreshData(dbMap *gorp.DbMap) {
	var numPages int = 1
	for i := 0; i <= numPages; i++ {
		var incidents []Incident
		incidents, numPages = getData(i)

		for _, incident := range incidents {
			key, _ := dbMap.SelectStr("select Id from Incident where id = ?", incident.Id)

			if key == "" {
				if err := dbMap.Insert(&incident); err != nil {
					fmt.Println(incident)
					panic(err)
				}
			}
		}
	}
}

func getData(page int) ([]Incident, int) {
	if rootNode, err := fetchPage(page); err != nil {
		return make([]Incident, 0), 0
	} else {
		tBody, _ := scrape.Find(rootNode, scrape.ByTag(atom.Tbody))
		if tBody == nil {
			return make([]Incident, 0), 0
		}

		rows := scrape.FindAll(tBody, scrape.ByTag(atom.Tr))

		incidents := make([]Incident, len(rows))
		for i, row := range rows {
			cols := scrape.FindAll(row, scrape.ByTag(atom.Td))
			incident := Incident{}
			for j, col := range cols {
				switch j {
				case 0:
					if date, err := time.Parse("January _2, 2006", col.FirstChild.Data); err == nil {
						incident.Date = date
					} else {
						fmt.Println(err)
					}
				case 1:
					incident.State = col.FirstChild.Data
				case 2:
					incident.City = col.FirstChild.Data
				case 3:
					incident.Address = col.FirstChild.Data
				case 4:
					if killed, err := strconv.ParseInt(col.FirstChild.Data, 10, 32); err == nil {
						incident.Killed = int(killed)
					}
				case 5:
					if injured, err := strconv.ParseInt(col.FirstChild.Data, 10, 32); err == nil {
						incident.Injured = int(injured)
					}
				case 6:
					atags := scrape.FindAll(col, scrape.ByTag(atom.A))
					if len(atags) == 2 {
						incidentPath := scrape.Attr(atags[0], "href")
						incident.Id = filepath.Base(incidentPath)
						incident.Source = scrape.Attr(atags[1], "href")
					}
				}
			}

			incidents[i] = incident
		}

		var lastPageNum int64
		if lastPage, _ := scrape.Find(rootNode, scrape.ByClass("pager-last")); lastPage != nil {
			path := scrape.Attr(lastPage.FirstChild, "href")
			numRegex := regexp.MustCompile("[^0-9]")
			path = numRegex.ReplaceAllString(path, "")
			lastPageNum, _ = strconv.ParseInt(path, 10, 64)
		}

		return incidents, int(lastPageNum)
	}
}

func fetchPage(page int) (rootNode *html.Node, err error) {
	url := base_url
	if page > 0 {
		url = fmt.Sprint(url, "?page=", page)
	}

	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return
	}

	defer resp.Body.Close()
	rootNode, err = html.Parse(resp.Body)

	return
}
