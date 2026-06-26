package cronjobservice

import (
	"fmt"
	"log"

	"github.com/emersion/go-sasl"
)

type xoauth2Client struct {
	username string
	token    string
}

func NewXOAuth2Client(
	username string,
	token string,
) sasl.Client {
	return &xoauth2Client{
		username: username,
		token:    token,
	}
}

func (a *xoauth2Client) Start() (
	mech string,
	ir []byte,
	err error,
) {
	mech = "XOAUTH2"

	payload := fmt.Sprintf(
		"user=%s\x01auth=Bearer %s\x01\x01",
		a.username,
		a.token,
	)

	log.Printf("XOAUTH2 USER=[%s]", a.username)
	log.Printf("XOAUTH2 PAYLOAD LEN=%d", len(payload))

	ir = []byte(payload)

	return
}

func (a *xoauth2Client) Next(challenge []byte) ([]byte, error) {
	log.Printf("XOAUTH2 CHALLENGE RAW=%q", challenge)
	log.Printf("XOAUTH2 CHALLENGE TEXT=%s", string(challenge))
	return []byte{}, nil
}
