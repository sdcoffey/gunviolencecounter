package track

import "time"

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
