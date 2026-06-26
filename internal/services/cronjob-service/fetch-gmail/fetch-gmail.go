package cronjobservice

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/services/cronjob"
	ticketService "prime-customer-care/internal/services/ticket-service"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"

	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"gorm.io/gorm"
)

const fetchLimit = 50

func init() {
	cronjob.RegisterJob("FetchEmail", FetchEmail, "*/1 * * * *")
}

// --- Config ---

type fetchConfig struct {
	IMAPHost     string
	IMAPPort     string
	IMAPUser     string
	IMAPPassword string

	TenantID     string
	ClientID     string
	ClientSecret string
}

func loadConfig() (*fetchConfig, error) {
	cfg := &fetchConfig{
		IMAPHost:     os.Getenv("IMAP_HOST"),
		IMAPPort:     os.Getenv("IMAP_PORT"),
		IMAPUser:     os.Getenv("IMAP_USER"),
		IMAPPassword: os.Getenv("IMAP_PASSWORD"),

		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
	}

	if cfg.IMAPHost == "" {
		return nil, fmt.Errorf("missing env: IMAP_HOST")
	}

	if cfg.IMAPPort == "" {
		return nil, fmt.Errorf("missing env: IMAP_PORT")
	}

	if cfg.IMAPUser == "" {
		return nil, fmt.Errorf("missing env: IMAP_USER")
	}

	hasOAuth :=
		cfg.TenantID != "" &&
			cfg.ClientID != "" &&
			cfg.ClientSecret != ""

	hasPassword := cfg.IMAPPassword != ""

	if !hasOAuth && !hasPassword {
		return nil, fmt.Errorf(
			"missing auth config: provide either OAuth or IMAP password",
		)
	}

	log.Printf("TenantID=[%s]", cfg.TenantID)
	log.Printf("ClientID=[%s]", cfg.ClientID)
	log.Printf("TenantID length=%d", len(cfg.TenantID))
	log.Printf("ClientID length=%d", len(cfg.ClientID))

	return cfg, nil
}

// --- State ---

type fetchState struct {
	LastUID   uint32
	SinceDate time.Time
}

func loadState(gormx *gorm.DB) (*fetchState, error) {
	lastUIDStr, err := getConfigValue(gormx, "fetch_email_last_uid")
	if err != nil {
		return nil, fmt.Errorf("fetch_email_last_uid: %w", err)
	}

	sinceDateStr, err := getConfigValue(gormx, "fetch_email_since_date")
	if err != nil {
		return nil, fmt.Errorf("fetch_email_since_date: %w", err)
	}

	lastUID64, err := strconv.ParseUint(lastUIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid fetch_email_last_uid value: %w", err)
	}

	sinceDate, err := time.Parse(time.RFC3339, sinceDateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid fetch_email_since_date value: %w", err)
	}

	return &fetchState{
		LastUID:   uint32(lastUID64),
		SinceDate: sinceDate,
	}, nil
}

func getConfigValue(gormx *gorm.DB, code string) (string, error) {
	var value string
	result := gormx.Table("system_config").
		Select("value").
		Where("config_code = ?", code).
		Scan(&value)

	if result.Error != nil {
		return "", fmt.Errorf("query failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("config_code '%s' not found", code)
	}

	return value, nil
}

func saveLastUID(gormx *gorm.DB, uid uint32) error {
	result := gormx.Table("system_config").
		Where("config_code = ?", "fetch_email_last_uid").
		Update("value", strconv.FormatUint(uint64(uid), 10))

	if result.Error != nil {
		return fmt.Errorf("failed to save last_uid: %w", result.Error)
	}

	return nil
}

// --- IMAP ---

type email struct {
	UID     uint32
	Subject string
	From    string
	Body    string
	Date    time.Time
}

func getAccessToken(cfg *fetchConfig) (string, error) {
	oauthCfg := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL: fmt.Sprintf(
			"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
			cfg.TenantID,
		),
		Scopes: []string{
			"https://outlook.office365.com/.default",
		},
	}

	token, err := oauthCfg.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("get token failed: %w", err)
	}

	return token.AccessToken, nil

}

func connectIMAP(cfg *fetchConfig) (*client.Client, error) {
	token, err := getAccessToken(cfg)
	if err != nil {
		return nil, err
	}

	log.Printf(
		"[FetchEmail] access token acquired len=%d",
		len(token),
	)

	log.Printf("IMAP_USER=[%s]", cfg.IMAPUser)

	parts := strings.Split(token, ".")

	if len(parts) >= 2 {
		payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
		log.Printf("JWT Payload=%s", string(payload))
	}

	addr := fmt.Sprintf("%s:%s", cfg.IMAPHost, cfg.IMAPPort)

	log.Printf("STEP 1: DialTLS [%s]", addr)

	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	c.SetDebug(os.Stdout)

	log.Printf("STEP 2: Capability")

	caps, err := c.Capability()
	log.Printf("CAPABILITY=%v", caps)
	log.Printf("CAPABILITY_ERR=%v", err)

	log.Printf("STEP 3: Authenticate")

	auth := NewXOAuth2Client(
		cfg.IMAPUser,
		token,
	)

	if err := c.Authenticate(auth); err != nil {
		log.Printf("STEP 3 FAILED")
		_ = c.Logout()
		return nil, fmt.Errorf("oauth authenticate failed: %w", err)
	}

	log.Printf("STEP 4: Select INBOX")

	if _, err := c.Select("INBOX", false); err != nil {
		log.Printf("STEP 4 FAILED")
		_ = c.Logout()
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	log.Printf("STEP 5: SUCCESS")

	return c, nil
}

func connectIMAPOld(cfg *fetchConfig) (*client.Client, error) {
	addr := fmt.Sprintf("%s:%s", cfg.IMAPHost, cfg.IMAPPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := c.Login(cfg.IMAPUser, cfg.IMAPPassword); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	if _, err := c.Select("INBOX", false); err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	return c, nil
}

func searchEmails(c *client.Client, state *fetchState) ([]uint32, error) {
	criteria := imap.NewSearchCriteria()
	criteria.Since = state.SinceDate

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(state.LastUID+1, ^uint32(0))
	criteria.Uid = seqSet

	uids, err := c.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Limit to fetchLimit
	if len(uids) > fetchLimit {
		uids = uids[:fetchLimit]
	}

	return uids, nil
}

func fetchEmails(c *client.Client, uids []uint32) ([]email, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	fetchItems := []imap.FetchItem{
		imap.FetchUid,
		imap.FetchEnvelope,
		"BODY.PEEK[]",
	}

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- c.UidFetch(seqSet, fetchItems, messages)
	}()

	var emails []email
	for msg := range messages {
		if msg == nil {
			continue
		}

		e := email{UID: msg.Uid}

		if msg.Envelope != nil {
			e.Subject = msg.Envelope.Subject
			e.Date = msg.Envelope.Date
			if len(msg.Envelope.From) > 0 && msg.Envelope.From[0] != nil {
				e.From = msg.Envelope.From[0].Address()
			}
		}

		body, err := parseBody(msg)
		if err != nil {
			log.Printf("failed to parse body for UID %d: %v", msg.Uid, err)
		}
		e.Body = body

		emails = append(emails, e)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return emails, nil
}

func parseBody(msg *imap.Message) (string, error) {
	var r io.Reader
	for _, value := range msg.Body {
		r = value
		break
	}
	if r == nil {
		return "", nil
	}

	raw, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	mr, err := mail.CreateReader(strings.NewReader(string(raw)))
	if err != nil {
		decoder := charmap.ISO8859_1.NewDecoder()
		decoded, _, err := transform.String(decoder, string(raw))
		if err != nil {
			return "", err
		}
		mr, err = mail.CreateReader(strings.NewReader(decoded))
		if err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch part.Header.(type) {
		case *mail.InlineHeader:
			body, err := io.ReadAll(part.Body)
			if err != nil {
				break
			}
			sb.Write(body)
		}
	}

	return sb.String(), nil
}

func markAsRead(c *client.Client, uids []uint32) error {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	flags := []interface{}{imap.SeenFlag}
	return c.UidStore(seqSet, imap.FormatFlagsOp(imap.AddFlags, true), flags, nil)
}

// --- Ticket ---

func createTicketsFromEmails(gormx *gorm.DB, emails []email) error {
	req := make([]ticketService.CreateTicketRequest, 0, len(emails))
	for _, e := range emails {
		req = append(req, ticketService.CreateTicketRequest{
			TicketChannel: "EMAIL",
			Email:         e.From,
			Status:        "PENDING",
		})
	}

	_, err := ticketService.CreateTickets(gormx, nil, req)
	if err != nil {
		return fmt.Errorf("failed to create tickets: %w", err)
	}

	return nil
}

// --- Main Job ---

func FetchEmail() {

	log.Printf("$$$ [FetchEmail] $$$")

	cfg, err := loadConfig()
	if err != nil {
		log.Printf("[FetchEmail] config error: %v", err)
		return
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		log.Printf("[FetchEmail] db connect error: %v", err)
		return
	}
	defer db.CloseGORM(gormx)

	state, err := loadState(gormx)
	if err != nil {
		log.Printf("[FetchEmail] load state error: %v", err)
		return
	}

	var imapConn *client.Client

	if os.Getenv("USE_OAUTH_IMAP") == "true" {
		imapConn, err = connectIMAP(cfg)
	} else {
		imapConn, err = connectIMAPOld(cfg)
	}

	if err != nil {
		log.Printf("[FetchEmail] imap connect error: %v", err)
		return
	}
	defer imapConn.Logout()

	uids, err := searchEmails(imapConn, state)
	if err != nil {
		log.Printf("[FetchEmail] search error: %v", err)
		return
	}

	if len(uids) == 0 {
		return
	}

	log.Printf("##### [FetchEmail] !!!!")

	emails, err := fetchEmails(imapConn, uids)
	if err != nil {
		log.Printf("[FetchEmail] fetch error: %v", err)
		return
	}

	// Filter emails by since_date
	var filtered []email
	for _, e := range emails {
		if e.Date.After(state.SinceDate) {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return
	}

	if err := createTicketsFromEmails(gormx, filtered); err != nil {
		log.Printf("[FetchEmail] create tickets error: %v", err)
		return
	}

	// collect UIDs from filtered only
	var filteredUIDs []uint32
	for _, e := range filtered {
		filteredUIDs = append(filteredUIDs, e.UID)
	}

	if err := markAsRead(imapConn, filteredUIDs); err != nil {
		log.Printf("[FetchEmail] mark as read error: %v", err)
	}

	var maxUID uint32
	for _, uid := range uids {
		if uid > maxUID {
			maxUID = uid
		}
	}

	if err := saveLastUID(gormx, maxUID); err != nil {
		log.Printf("[FetchEmail] save state error: %v", err)
		return
	}
}
