package scraper

import (
	"crypto/tls"
	"database/sql"
	"log"

	"github.com/Cfirth725/parcel-herder/internal/db"
	"github.com/Cfirth725/parcel-herder/internal/parser"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// FetchAndProcessMailboxes connects to a target secure IMAP server, polls for unread
// logistics messages, extracts relevant plaintext chunks, and routes parsed data to storage.
func FetchAndProcessMailboxes(server, email, password string, database *sql.DB) error {
	log.Printf("[SYNC] Connecting to secure IMAP stream for: %s...", email)

	options := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	client, err := imapclient.DialTLS(server, options)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Login(email, password).Wait(); err != nil {
		return err
	}
	log.Printf("[SECURE] Authenticated successfully for: %s", email)

	mbox, err := client.Select("Logistics", nil).Wait()
	if err != nil {
		return err
	}

	if mbox.NumMessages == 0 {
		log.Printf("[SYNC] INBOX is entirely empty for %s.", email)
		return nil
	}

	criteria := &imap.SearchCriteria{
		Not: []imap.SearchCriteria{{Flag: []imap.Flag{imap.FlagSeen}}},
	}

	searchCmd := client.Search(criteria, nil)
	searchRes, err := searchCmd.Wait()
	if err != nil {
		return err
	}

	seqNums := searchRes.AllSeqNums()
	if len(seqNums) == 0 {
		log.Printf("[SYNC] No new unread logistics emails found for %s.", email)
		return nil
	}

	log.Printf("[SYNC] Found %d unread emails to parse...", len(seqNums))

	bodySection := &imap.FetchItemBodySection{Specifier: imap.PartSpecifierText}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	fetchCmd := client.Fetch(searchRes.All, fetchOptions)
	defer fetchCmd.Close()

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		msgData, err := msg.Collect()
		if err != nil {
			log.Printf("[ERROR] Failed to collect message sections: %v", err)
			continue
		}

		if rawBody := msgData.FindBodySection(bodySection); rawBody != nil {
			bodyStr := string(rawBody)

			payload := parser.ParseEmailBody(bodyStr)

			if payload.TrackingNumber != "" || payload.IsLockerToken {
				accountID, err := db.GetAccountID(database, email)
				if err != nil {
					log.Printf("[ERROR] Database lookup failed for email hash: %v", err)
					continue
				}

				isLockerInt := 0
				if payload.IsLockerToken {
					isLockerInt = 1
				}

				err = db.InsertOrUpdatePackage(database, accountID, payload.TrackingNumber, payload.Carrier, payload.LockerCode, isLockerInt)
				if err != nil {
					log.Printf("[ERROR] Failed to persist package telemetry to DB: %v", err)
				} else {
					log.Printf("[OK] Routed %s telemetry to Account ID %d", payload.Carrier, accountID)
				}
			}
		}
	}
	return nil
}
