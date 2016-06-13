package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/github.com/go-gorp/gorp"
	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/github.com/gorilla/mux"
	_ "github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/github.com/yhat/scrape"
	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/golang.org/x/net/html"
	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/golang.org/x/net/html/atom"
	"github.com/sdcoffey/gunviolencecounter/sunlight_api"
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
	Name  string
	Email string
	ZIP   string
	State string
}

var count int64

func main() {
	dbMap := initDb()
	go refreshData(dbMap)
	go refreshLoop(6*time.Hour, func() {
		refreshData(dbMap)
		updateCount(dbMap)
	})

	updateCount(dbMap)

	r := mux.NewRouter()
	r.HandleFunc("/v1/email", func(writer http.ResponseWriter, req *http.Request) {
		addEmail(writer, req, dbMap)
	}).Methods("POST")
	r.HandleFunc("/v1/email", func(writer http.ResponseWriter, req *http.Request) {
		listEmails(writer, req, dbMap)
	})
	r.HandleFunc("/v1/reps", func(writer http.ResponseWriter, req *http.Request) {
		listReps(writer, req, dbMap)
	})
	r.HandleFunc("/v1/victimCount", func(writer http.ResponseWriter, req *http.Request) {
		getCount(writer)
	}).Methods("GET")

	fmt.Println("Listening")
	http.ListenAndServe(":3001", r)
}

func getCount(writer http.ResponseWriter) {
	writer.WriteHeader(200)
	writer.Write([]byte(fmt.Sprint(count)))
}

func updateCount(dbmap *gorp.DbMap) {
	if num, err := dbmap.SelectInt("select sum(killed + injured) as thesum from Incident where Date > timestamp '2016-01-01'"); err == nil {
		count = num
	}
}

func listReps(writer http.ResponseWriter, req *http.Request, dbmap *gorp.DbMap) {
	if req.Header.Get("Authorization") != os.Getenv("EMAIL_PW") {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	var reps []sunlight_api.Rep
	dbmap.Select(&reps, "select * from Rep")
	encoder := json.NewEncoder(writer)
	writer.Header().Add("Content-Type", "application/json")
	encoder.Encode(reps)
}

func listEmails(writer http.ResponseWriter, req *http.Request, dbmap *gorp.DbMap) {
	if req.Header.Get("Authorization") != os.Getenv("EMAIL_PW") {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	var emails []Email
	dbmap.Select(&emails, "select * from Email")
	encoder := json.NewEncoder(writer)
	writer.Header().Add("Content-Type", "application/json")
	encoder.Encode(emails)
}

func addEmail(writer http.ResponseWriter, req *http.Request, dbMap *gorp.DbMap) {
	body, _ := httputil.DumpRequest(req, true)
	fmt.Println("/email", string(body))

	var submission Email
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&submission); err != nil {
		writer.WriteHeader(400)
	} else if strings.Contains(submission.Email, "@") && strings.Contains(submission.Email, ".") && submission.ZIP != "" {
		email := Email{Name: submission.Name, Email: submission.Email, ZIP: submission.ZIP}
		if key, _ := dbMap.SelectStr("select Email from Email where Email = $1 AND ZIP = $2", email.Email, email.ZIP); key == "" {
			reps := sunlight_api.GetReps(submission.ZIP)
			if len(reps) > 0 {
				email.State = reps[0].State
			}
			fmt.Println("Adding user", email)
			dbMap.Insert(&email)
			for _, rep := range reps {
				fmt.Println("Adding rep", rep)
				dbMap.Insert(&rep)
			}
			writer.WriteHeader(200)
			fmt.Println("Inserted", email)
		} else {
			fmt.Println("Duplicate", email)
		}
	} else {
		writer.WriteHeader(400)
	}
}

func initDb() *gorp.DbMap {
	dbinfo := fmt.Sprintf("user=docker password=%s dbname=docker sslmode=disable host=db", os.Getenv("GV_PG_PASS"))
	if db, err := sql.Open("postgres", dbinfo); err != nil {
		panic(err)
	} else if err = db.Ping(); err != nil {
		panic(err)
	} else {
		fmt.Println("Connected to Postgres")
		dbMap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
		dbMap.AddTable(Incident{}).SetKeys(false, "Id")
		dbMap.AddTable(Email{}).SetKeys(false, "Email")
		dbMap.AddTable(sunlight_api.Rep{}).SetKeys(false, "BioGuideId")
		dbMap.CreateTablesIfNotExists()
		return dbMap
	}
}

func refreshLoop(d time.Duration, updateFunc func()) {
	refreshTimer := time.Tick(d)
	for range refreshTimer {
		updateFunc()
	}
}

func refreshData(dbMap *gorp.DbMap) {
	var numPages int = 1
	for i := 0; i <= numPages; i++ {
		var incidents []Incident
		incidents, numPages = getData(i)

		for _, incident := range incidents {
			existingIncident := getIncident(incident.Id, dbMap)

			if existingIncident.Id == "" {
				fmt.Println("Adding incident", incident)
				if err := dbMap.Insert(&incident); err != nil {
					fmt.Println("ERROR - adding incident", incident, err.Error())
				}
			} else if existingIncident.Injured != incident.Injured || existingIncident.Killed != incident.Killed {
				fmt.Println("Updating incident", incident.Id)
				dbMap.Update(incident)
			}
		}
	}
}

func getIncident(id string, dbMap *gorp.DbMap) Incident {
	var incident Incident
	dbMap.SelectOne(&incident, "select * from Incident where id = $1", id)
	return incident
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
