package cronjobservice

import (
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
	imapClient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"gorm.io/gorm"
)

const fetchLimit = 50

func init() {
	cronjob.RegisterJob("FetchEmail", FetchEmail, "*/5 * * * *")
}

// --- Config ---

type fetchConfig struct {
	IMAPHost     string
	IMAPPort     string
	IMAPUser     string
	IMAPPassword string
}

func loadConfig() (*fetchConfig, error) {
	cfg := &fetchConfig{
		IMAPHost:     os.Getenv("IMAP_HOST"),
		IMAPPort:     os.Getenv("IMAP_PORT"),
		IMAPUser:     os.Getenv("IMAP_USER"),
		IMAPPassword: os.Getenv("IMAP_PASSWORD"),
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
	if cfg.IMAPPassword == "" {
		return nil, fmt.Errorf("missing env: IMAP_PASSWORD")
	}

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
}

func connectIMAP(cfg *fetchConfig) (*imapClient.Client, error) {
	addr := fmt.Sprintf("%s:%s", cfg.IMAPHost, cfg.IMAPPort)
	c, err := imapClient.DialTLS(addr, nil)
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

func searchEmails(c *imapClient.Client, state *fetchState) ([]uint32, error) {
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

func fetchEmails(c *imapClient.Client, uids []uint32) ([]email, error) {
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

func markAsRead(c *imapClient.Client, uids []uint32) error {
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
	cfg, err := loadConfig()
	if err != nil {
		log.Printf("[FetchEmail] config error: %v", err)
		return
	}

	gormx, err := db.ConnectGORM(os.Getenv("DATABASE_URL"))
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

	log.Printf("[FetchEmail] last_uid: %d, since: %s", state.LastUID, state.SinceDate.Format(time.RFC3339))

	imapConn, err := connectIMAP(cfg)
	if err != nil {
		log.Printf("[FetchEmail] imap connect error: %v", err)
		return
	}
	defer imapConn.Terminate()

	uids, err := searchEmails(imapConn, state)
	if err != nil {
		log.Printf("[FetchEmail] search error: %v", err)
		return
	}

	if len(uids) == 0 {
		log.Println("[FetchEmail] no new emails")
		return
	}

	log.Printf("[FetchEmail] found %d emails", len(uids))

	emails, err := fetchEmails(imapConn, uids)
	if err != nil {
		log.Printf("[FetchEmail] fetch error: %v", err)
		return
	}

	if err := createTicketsFromEmails(gormx, emails); err != nil {
		log.Printf("[FetchEmail] create tickets error: %v", err)
		return
	}

	if err := markAsRead(imapConn, uids); err != nil {
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

	log.Printf("[FetchEmail] success — saved last_uid: %d", maxUID)
}
