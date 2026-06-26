package cronjobservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/services/cronjob"
	ticketService "prime-customer-care/internal/services/ticket-service"

	"golang.org/x/oauth2/clientcredentials"
	"gorm.io/gorm"
)

const fetchLimit = 50

func init() {
	cronjob.RegisterJob("FetchEmail", FetchEmail, "*/1 * * * *")
}

// --- Config ---

type fetchConfig struct {
	IMAPUser     string // ใช้เป็น Target Mailbox สำหรับส่งให้ Graph API
	TenantID     string
	ClientID     string
	ClientSecret string
}

type GraphMessageResponse struct {
	Value []GraphMessage `json:"value"`
}

type GraphMessage struct {
	ID               string          `json:"id"`
	Subject          string          `json:"subject"`
	ReceivedDateTime time.Time       `json:"receivedDateTime"`
	From             GraphFromSector `json:"from"`
	Body             GraphBodySector `json:"body"`
}

type GraphFromSector struct {
	EmailAddress GraphEmailAddress `json:"emailAddress"`
}

type GraphEmailAddress struct {
	Address string `json:"address"`
}

type GraphBodySector struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

func loadConfig() (*fetchConfig, error) {
	cfg := &fetchConfig{
		IMAPUser:     os.Getenv("IMAP_USER"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
	}

	if cfg.IMAPUser == "" {
		return nil, fmt.Errorf("missing env: IMAP_USER")
	}

	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("missing Azure OAuth configuration in env")
	}

	log.Printf("TenantID=[%s]", cfg.TenantID)
	log.Printf("ClientID=[%s]", cfg.ClientID)
	log.Printf("TenantID length=%d", len(cfg.TenantID))
	log.Printf("ClientID length=%d", len(cfg.ClientID))

	return cfg, nil
}

type email struct {
	UID     string
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
			"https://graph.microsoft.com/.default",
		},
	}

	token, err := oauthCfg.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("get token failed: %w", err)
	}

	return token.AccessToken, nil
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
	log.Printf("$$$ [FetchEmail via Microsoft Graph API] $$$")

	// 1. โหลดคอนฟิกจาก Environment (.env)
	cfg, err := loadConfig()
	if err != nil {
		log.Printf("[FetchEmail] config error: %v", err)
		return
	}

	// 2. เชื่อมต่อฐานข้อมูลสำหรับระบบตั๋ว
	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		log.Printf("[FetchEmail] db connect error: %v", err)
		return
	}
	defer db.CloseGORM(gormx)

	// 3. ดึง Access Token สำหรับ Microsoft Graph
	token, err := getAccessToken(cfg)
	if err != nil {
		log.Printf("[FetchEmail] failed to get token: %v", err)
		return
	}

	// 4. ดึงอีเมลที่ยังไม่ได้อ่าน (Unread) ผ่าน HTTP GET Request
	emails, err := fetchEmailsFromGraph(token, cfg.IMAPUser)
	if err != nil {
		log.Printf("[FetchEmail] fetch error: %v", err)
		return
	}

	if len(emails) == 0 {
		log.Printf("[FetchEmail] no new unread emails found.")
		return
	}

	log.Printf("[FetchEmail] found %d new emails. processing...", len(emails))

	// 5. ส่งรายชื่ออีเมลเข้าระบบเพื่อสร้าง Ticket
	if err := createTicketsFromEmails(gormx, emails); err != nil {
		log.Printf("[FetchEmail] create tickets error: %v", err)
		return
	}

	// 6. วนลูปสั่งอัปเดตสถานะใน Microsoft เป็น "อ่านแล้ว (Read)" เพื่อป้องกันการดึงซ้ำ
	for _, e := range emails {
		if err := markAsReadInGraph(token, cfg.IMAPUser, e.UID); err != nil {
			log.Printf("[FetchEmail] failed to mark message %s as read: %v", e.UID, err)
		} else {
			log.Printf("[FetchEmail] successfully marked message %s as read.", e.UID)
		}
	}

	log.Printf("$$$ [FetchEmail] completed successfully $$$")
}

func fetchEmailsFromGraph(accessToken string, targetUser string) ([]email, error) {
	apiURL := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/mailFolders/inbox/messages?$filter=isRead+eq+false&$top=%d&$select=id,subject,receivedDateTime,from,body",
		targetUser,
		fetchLimit,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph api returned status: %s", resp.Status)
	}

	var graphResp GraphMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphResp); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	var emails []email
	for _, msg := range graphResp.Value {
		e := email{
			UID:     msg.ID,
			Subject: msg.Subject,
			Date:    msg.ReceivedDateTime,
			From:    msg.From.EmailAddress.Address,
			Body:    msg.Body.Content,
		}
		emails = append(emails, e)
	}

	return emails, nil
}

func markAsReadInGraph(accessToken string, targetUser string, messageID string) error {
	apiURL := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/messages/%s",
		targetUser,
		messageID,
	)

	bodyMap := map[string]interface{}{
		"isRead": true,
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create patch request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("patch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mark as read, status: %s", resp.Status)
	}

	return nil
}
