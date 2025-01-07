package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/*
A few usage examples:

post event:

curl --header "Content-Type: application/json" \
--request POST \
--data "[{\"property\":\"test\",\"ip\":\"192.168.0.0\",\"user_agent\":\"secret agent\",\"description\":\"awesome thing\"}]" \
http://localhost:8091/events

fetch quick stats:

curl "http://localhost:8091/stats?property=test"

*/

const file string = "tracking.db" // name of database (sqlite3)
const listenAddr string = ":8091" // address/port to listen at
const eventsTableCreate string = `
  CREATE TABLE IF NOT EXISTS events (
  id INTEGER NOT NULL PRIMARY KEY,
  timestamp INTEGER, 
  property TEXT,
  ip TEXT,
  user_agent TEXT,
  description TEXT
  );`
const indexOnTimestamp string = `CREATE INDEX IF NOT EXISTS idx_property_timestamp ON events (property,timestamp)`
const uniqueStatsSql string = `select count(distinct(ip)) from events where property = ? AND timestamp > ?`
const hitStatsSql string = `select count(id) from events where property = ? AND timestamp > ?`
const insertSql string = `INSERT INTO events VALUES(NULL,?,?,?,?,?)`

func main() {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(eventsTableCreate); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec(indexOnTimestamp); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/events", postEventHandler)
	http.HandleFunc("/stats", statsHandler)
	http.ListenAndServe(listenAddr, nil)
}

type Event struct {
	Property    string  `json:"property"`
	Ip          *string `json:"ip"`
	UserAgent   *string `json:"user_agent"`
	Description *string `json:"description"`
}

func postEventHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		return
	}

	decoder := json.NewDecoder(req.Body)
	var events []Event
	err := decoder.Decode(&events)
	if err != nil {
		log.Println("unable to parse!")
		return
	}

	eventsPopulated := Map(events, func(event Event) Event {

		// populate ip if not provided
		if event.Ip == nil {
			ip := req.RemoteAddr
			// remove remote port from ip
			lastColonIndex := strings.LastIndex(ip, ":")
			if lastColonIndex > -1 {
				ip = string(ip[:lastColonIndex])
			}
			event.Ip = &ip
		}

		// populate user agent if not provided
		if event.UserAgent == nil {
			ua := req.UserAgent()
			event.UserAgent = &ua
		}

		// populate description if not provided
		if event.Description == nil {
			event.Description = &req.RequestURI
		}

		return event

	})

	logEvent(eventsPopulated)
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func statsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		return
	}

	if data, ok := req.URL.Query()["property"]; ok {
		property := data[0]
		if len(property) == 0 {
			return
		}

		db, err := sql.Open("sqlite3", file)
		if err != nil {
			log.Println(err)
			return
		}
		defer db.Close()

		timestamp := time.Now().Unix() - 86400*30
		lastMonthUniquesRow := db.QueryRow(uniqueStatsSql, property, timestamp)
		lastMonthHitsRow := db.QueryRow(hitStatsSql, property, timestamp)
		timestamp = time.Now().Unix() - 86400
		lastDayUniquesRow := db.QueryRow(uniqueStatsSql, property, timestamp)
		lastDayHitsRow := db.QueryRow(hitStatsSql, property, timestamp)

		stats := struct {
			Version             string `json:"version"`
			LastMonthUniqueHits int    `json:"last_month_unique_hits"`
			LastDayUniqueHits   int    `json:"last_day_unique_hits"`
			LastMonthHits       int    `json:"last_month_hits"`
			LastDayHits         int    `json:"last_day_hits"`
		}{
			Version: "0.0.3",
		}

		_ = lastMonthUniquesRow.Scan(&stats.LastMonthUniqueHits)
		_ = lastDayUniquesRow.Scan(&stats.LastDayUniqueHits)
		_ = lastMonthHitsRow.Scan(&stats.LastMonthHits)
		_ = lastDayHitsRow.Scan(&stats.LastDayHits)

		b, err := json.Marshal(stats)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprint(w, string(b))

	}
}

func logEvent(events []Event) {
	time := time.Now().Unix()

	// insert db and autoclose!
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	for i := 0; i < len(events); i++ {
		event := events[i]
		_, err = db.Exec(insertSql, time, event.Property, *event.Ip, *event.UserAgent, *event.Description)
		if err != nil {
			log.Println(err)
		}
	}
}
