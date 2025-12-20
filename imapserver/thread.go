package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SessionThread is an IMAP session which supports the THREAD extension (RFC 5256).
//
// This extension allows clients to retrieve messages organized into conversation threads.
type SessionThread interface {
	Session

	// Thread searches the mailbox and returns messages organized into threads.
	// algorithm specifies the threading algorithm (ORDEREDSUBJECT or REFERENCES).
	// charset specifies the character set for the search criteria.
	// criteria contains the search criteria to filter messages.
	Thread(kind NumKind, algorithm imap.ThreadAlgorithm, charset string, criteria *imap.SearchCriteria) (*imap.ThreadData, error)
}

func (c *Conn) handleThread(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Parse threading algorithm
	var algorithmStr string
	if !dec.ExpectAtom(&algorithmStr) {
		return dec.Err()
	}

	algorithm := imap.ThreadAlgorithm(strings.ToUpper(algorithmStr))
	switch algorithm {
	case imap.ThreadOrderedSubject, imap.ThreadReferences:
		// Valid algorithms
	default:
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: fmt.Sprintf("Unknown threading algorithm: %s", algorithmStr),
		}
	}

	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Parse charset
	var charset string
	if !dec.ExpectAString(&charset) {
		return dec.Err()
	}

	// Validate charset
	switch strings.ToUpper(charset) {
	case "US-ASCII", "UTF-8":
		// supported charsets
	default:
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeBadCharset,
			Text: "Only US-ASCII and UTF-8 are supported THREAD charsets",
		}
	}

	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Parse search criteria (same as SEARCH command)
	var searchCriteria imap.SearchCriteria
	for {
		if err := readSearchKey(&searchCriteria, dec); err != nil {
			return fmt.Errorf("in search-key: %w", err)
		}

		if !dec.SP() {
			break
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	// Check if session supports THREAD
	sessionThread, ok := c.session.(SessionThread)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "THREAD extension not supported",
		}
	}

	// Call the session's Thread method
	data, err := sessionThread.Thread(numKind, algorithm, charset, &searchCriteria)
	if err != nil {
		return err
	}

	// Write THREAD response
	return c.writeThread(data, numKind)
}

func (c *Conn) writeThread(data *imap.ThreadData, numKind NumKind) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("THREAD")

	// Write each top-level thread
	for _, thread := range data.Threads {
		enc.SP()
		c.writeThreadNode(enc, &thread, numKind)
	}

	return enc.CRLF()
}

func (c *Conn) writeThreadNode(enc *responseEncoder, node *imap.ThreadNode, numKind NumKind) {
	enc.Special('(')

	if node.Num > 0 {
		if numKind == NumKindUID {
			enc.UID(imap.UID(node.Num))
		} else {
			enc.Number(node.Num)
		}
	}

	// Write children
	for i, child := range node.Children {
		// Add space before child if we have a number or previous children
		if node.Num > 0 || i > 0 {
			enc.SP()
		}
		c.writeThreadNode(enc, &child, numKind)
	}

	enc.Special(')')
}
