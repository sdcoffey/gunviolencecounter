package email

import (
	"github.com/sdcoffey/gunviolencecounter/sunlight_api"
	"net/smtp"
	"text/template"
	"fmt"
	"github.com/go-gorp/gorp"
	"io/ioutil"
	"bytes"
	"github.com/sdcoffey/gunviolencecounter/scrape"
)

var (
	emailBodyTemplate *template.Template
)

func init() {
	emailBodyTemplate = emailBodyTemplate.New("Email Body")
	bodyDat, err := ioutil.ReadFile("email_template")
	if err != nil {
		panic(err)
	}

	emailBodyTemplate.Parse(string(bodyDat))
}

func SendMailForIncident(i track.Incident, db *gorp.DbMap) error {
	var senders []track.Email
	db.Select(&senders, "Select * from Email")
	for _, sender := range senders {

	}
}

func sendMail(senders []string, rep sunlight_api.Rep) {
	if c, err := smtp.Dial("mail:25"); err != nil {
		return err
	} else {
		c.Mail(fmt.Sprintf("help+%s@stopgunviolence.today", rep.District))
		c.Rcpt(rep.Email)
		fmt.Fprint(c.Data(), renderBody(senders, db, rep))
		if err := c.Close(); err != nil {
			return err
		} else if err = c.Quit(); err != nil {
			return err
		}
	}
	return nil
}

func renderBody(senders []string, db *gorp.DbMap, rep sunlight_api.Rep) (string, error) {
	var templateData struct {
		Rep sunlight_api.Rep
		Senders []string
		VictimCount int
		Incident string // todo pass incident
	}

	buf := bytes.NewBufferString("")
	emailBodyTemplate.Execute(buf, templateData)
	return buf.String()
}
