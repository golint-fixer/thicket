package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"
)

var messageTemplate *template.Template

func init() {
	messageTemplate = template.Must(template.New("message").Parse(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
	<s:Header>
		<o:Security s:mustUnderstand="1" xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
			<u:Timestamp u:Id="_0">
				<u:Created>{{.Created}}</u:Created>
				<u:Expires>{{.Expires}}</u:Expires>
			</u:Timestamp>
			<o:UsernameToken u:Id="uuid-1de0ecf4-477e-48d8-a40c-07d57d1c30da-1">
				<o:Username>{{.Username}}</o:Username>
				<o:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">{{.Password}}</o:Password>
			</o:UsernameToken>
		</o:Security>
	</s:Header>
	<s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
		<GetTransitionOfCareCcda xmlns="http://emr.ws.cp.gehcit.com/">
			<patientId>{{.PatientID}}</patientId>
			<documentId>{{.DocumentID}}</documentId>
			<providerId>{{.ProviderID}}</providerId>
			<orderId>{{.OrderID}}</orderId>
		</GetTransitionOfCareCcda>
	</s:Body>
</s:Envelope>`))
}

type envelope struct {
	Body body
}

type body struct {
	Response response `xml:"GetTransitionOfCareCcdaResponse"`
}

type response struct {
	Data data `xml:"return"`
}

type data struct {
	XML string `xml:"transitionOfCareAsXML"`
}

type apiSettings struct {
	Host            string
	Username        string
	Password        string
	Port            int
	DatabaseName    string
	IgnoreSSLErrors bool
}

type api struct {
	client   *http.Client
	Username string
	password string
	url      string
}

func newAPI(settings apiSettings) api {
	url := fmt.Sprintf("https://%s:%d/%s/ws/Services/emr", settings.Host, settings.Port, settings.DatabaseName)

	api := api{
		Username: settings.Username,
		password: settings.Password,
		url:      url,
	}

	if settings.IgnoreSSLErrors {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		api.client = &http.Client{Transport: tr}
	} else {
		api.client = &http.Client{}
	}

	return api
}

func (a *api) generateCcda(oid, pid, sdid, pvid int64) (string, error) {
	var message bytes.Buffer

	err := messageTemplate.Execute(&message, struct {
		Created    string
		Expires    string
		Username   string
		Password   string
		PatientID  int64
		DocumentID int64
		ProviderID int64
		OrderID    int64
	}{
		time.Now().Format(time.RFC3339) + "Z",
		time.Now().Add(5*time.Minute).Format(time.RFC3339) + "Z",
		a.Username,
		a.password,
		pid,
		sdid,
		pvid,
		oid,
	})

	if err != nil {
		return "", err
	}

	resp, err := a.client.Post(a.url, "text/xml; charset=utf-8", &message)

	if err != nil {
		return "", err
	}

	defer func() { _ = resp.Body.Close() }()

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	in := string(b)
	parser := xml.NewDecoder(bytes.NewBufferString(in))
	env := new(envelope)

	err = parser.DecodeElement(&env, nil)

	if err != nil {
		return "", err
	}

	ccda, err := base64.StdEncoding.DecodeString(env.Body.Response.Data.XML)

	if err != nil {
		return "", err
	}

	return string(ccda), nil
}
