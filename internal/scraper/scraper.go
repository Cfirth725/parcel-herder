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

// FetchAndProcessMailboxes connects to the IMAP target server using v2 mechanics.
func FetchAndProcessMailboxes(server, email, password string, database *sql.DB) error {
	log.Printf("Connecting to secure IMAP stream for: %s...", email)

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
	log.Printf("Authenticated successfully for: %s", email)

	mbox, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		return err
	}

	if mbox.NumMessages == 0 {
		log.Printf("INBOX is entirely empty for %s.", email)
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

	// FIX: Use the native v2 AllSeqNums() method to extract our sequence values
	seqNums := searchRes.AllSeqNums()
	if len(seqNums) == 0 {
		log.Printf("😴 No new unread logistics emails found for %s.", email)
		return nil
	}

	log.Printf("📬 Found %d unread emails to parse...", len(seqNums))

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
			log.Printf("FAILURE: Failed to collect message sections: %v", err)
			continue
		}

		if rawBody := msgData.FindBodySection(bodySection); rawBody != nil {
			bodyStr := string(rawBody)

			payload := parser.ParseEmailBody(bodyStr)

			if payload.TrackingNumber != "" || payload.IsLockerToken {
				accountID, _ := db.GetAccountID(database, email)
				log.Printf("Match Found! Routing %s data to Account ID %d", payload.Carrier, accountID)
			}
		}
	}

	return nil
}
