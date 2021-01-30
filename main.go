// Package providing a simple command line utility to inspect
// all emails in a given mailbox and to remove all duplicates.
//
// Note: When running, make sure that the imap server is set
// to move messages to bin or delete when message is
// marked as deleted over imap.
package main

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func main() {
	username := flag.String("username", "", "IMAP user (required)")
	password := flag.String("password", "", "IMAP password (required)")
	server := flag.String("server", "", "IMAP server (required)")
	mbox := flag.String("mbox", "", "Mailbox to remove duplicates from (required)")
	listOnlyDups := flag.Bool("list-only-dups", false, "If present, only duplicated messages are output")
	ignoreMessageID := flag.Bool("ignore-message-id", false, "If present, MessageId is ignored, a hash for each message is instead calculated")
	dryRun := flag.Bool("dry-run", false, "If present, no removal will be performed")
	flag.Parse()

	if *username == "" || *password == "" || *server == "" || *mbox == "" {
		flag.Usage()
		return
	}

	port := 0
	useTLS := true
	useStartTLS := false

	// Set default port
	if port == 0 {
		port = 143
		if useTLS {
			port = 993
		}
	}

	connectionString := fmt.Sprintf("%s:%d", *server, port)
	tlsConfig := &tls.Config{ServerName: *server}
	var c *client.Client
	var err error
	if useTLS {
		c, err = client.DialTLS(connectionString, tlsConfig)
	} else {
		c, err = client.Dial(connectionString)
	}

	if err != nil {
		panic(err)
	}
	// Start a TLS session
	if useStartTLS {
		if err = c.StartTLS(tlsConfig); err != nil {
			panic(err)
		}
	}

	err = c.Login(*username, *password)
	if err != nil {
		panic(err)
	}
	defer c.Logout()

	uids, err := FindDups(c, *mbox, *ignoreMessageID, *listOnlyDups)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find duplicates: %s\n", err)
		return
	}

	if !*dryRun {
		fmt.Println("will remove", len(uids), "messages")
		err = RemoveDups(c, *mbox, uids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot find duplicates: %s\n", err)
			return
		}
		fmt.Println("done")
	} else {
		fmt.Println("would have removed", len(uids), "messages")
	}

}

func FindDups(c *client.Client, mbox string, ignoreMessageID bool, listOnlyDups bool) (uids []uint32, err error) {
	st, err := c.Select(mbox, false)
	if err != nil {
		return nil, err
	}

	fmt.Println("MBOX UID", st.UidValidity)

	seqset := &imap.SeqSet{}
	seqset.AddRange(1, math.MaxUint32)

	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope}
	msgChan := make(chan *imap.Message, 1000)
	errChan := make(chan error, 1)
	go func() {
		err = c.UidFetch(seqset, items, msgChan)
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	uniqueIDs := make(map[string]struct{})
	var dups []uint32

	for msg := range msgChan {
		messageID := msg.Envelope.MessageId

		// instead hash the message contents
		if ignoreMessageID {
			messageID = ""
		}

		if messageID == "" {
			hash := sha1.New()
			builder := strings.Builder{}
			builder.WriteString("date:")
			builder.WriteString(msg.Envelope.Date.String())
			builder.WriteString("\nsubject:")
			builder.WriteString(msg.Envelope.Subject)
			for _, f := range msg.Envelope.From {
				builder.WriteString("\nfrom:")
				builder.WriteString(f.Address())
			}
			for _, f := range msg.Envelope.Sender {
				builder.WriteString("\nsender:")
				builder.WriteString(f.Address())
			}
			for _, f := range msg.Envelope.ReplyTo {
				builder.WriteString("\nreply-to:")
				builder.WriteString(f.Address())
			}
			for _, f := range msg.Envelope.To {
				builder.WriteString("\nto:")
				builder.WriteString(f.Address())
			}
			for _, f := range msg.Envelope.Cc {
				builder.WriteString("\ncc:")
				builder.WriteString(f.Address())
			}
			for _, f := range msg.Envelope.Bcc {
				builder.WriteString("\nbcc:")
				builder.WriteString(f.Address())
			}
			builder.WriteString("\nin-reply-to:")
			builder.WriteString(msg.Envelope.InReplyTo)
			messageID = base64.StdEncoding.EncodeToString(hash.Sum([]byte(builder.String())))
		}

		if !listOnlyDups {
			fmt.Printf("%s: %s %d %s:", mbox, msg.Envelope.Subject, msg.Uid, messageID)
		}
		if _, found := uniqueIDs[messageID]; found {
			dups = append(dups, msg.Uid)
			if listOnlyDups {
				fmt.Printf("%s: %s %d %s:", mbox, msg.Envelope.Subject, msg.Uid, messageID)
			}
			fmt.Println("duplicate")
			if listOnlyDups {
				fmt.Println("")
			}
			continue
		}
		if !listOnlyDups {
			fmt.Println("")
		}
		uniqueIDs[messageID] = struct{}{}
	}
	err = <-errChan
	return dups, err
}

func RemoveDups(c *client.Client, mbox string, uids []uint32) (err error) {
	_, err = c.Select(mbox, false)
	if err != nil {
		return err
	}

	for _, uid := range uids {
		seqSet := &imap.SeqSet{}
		seqSet.AddNum(uid)
		err = c.UidStore(seqSet, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.DeletedFlag}, nil)
		if err != nil {
			return err
		}
	}

	return c.Expunge(nil)
}
