package sunlight_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Rep struct {
	BioGuideId string `json:"bioguide_id"`
	Chamber    string `json:"chamber"`
	District   int    `json:"district"`
	FacebookId string `json:"facebook_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	LisId      string `json:"lis_id"`
	Email      string `json:"oc_email"`
	Phone      string `json:"phone"`
	State      string `json:"state"`
	Title      string `json:"title"`
	Twitter    string `json:"twitter_id"`
	Website    string `json:"website"`
	Party      string `json:"party"`
}

func GetReps(zip string) []Rep {
	url := fmt.Sprintf("https://congress.api.sunlightfoundation.com/legislators/locate?zip=%s&apikey=%s", zip, os.Getenv("SUNLIGHT_API_KEY"))
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		var results struct {
			Reps []Rep `json:"results"`
		}
		decoder.Decode(&results)
		reps := results.Reps
		for i, rep := range reps {
			rep.Email = strings.Replace(rep.Email, "opencongress.org", "emailcongress.us", 1)
			reps[i] = rep
		}
		return reps
	} else {
		panic(err)
	}

	return []Rep{}
}
