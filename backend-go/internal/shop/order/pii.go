package order

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/ritme/backend-go/internal/shop/store"
)

// Buyer PII at rest (user decision 2026-10-12, migration 00091). The identifying fields of a saved address (label,
// recipient name and phone, address line, postal code) and of an order's recipient snapshot (name, phone, address
// text, postal code) live in ONE sealed blob `pii_enc` — the PRIVATE_NOTE_KEY AES-256-GCM box (*loss.NoteBox), the
// B-N10-04 partner lab request pattern — bound to the owner (additional data "shop-address:<uid>" /
// "shop-order:<uid>"), so a blob copied onto another user's row does not open. city / province / is_default stay
// plain. New and edited rows write only the blob and clear the plaintext columns; a legacy row (no blob) is read from
// its plaintext columns. Without a usable key (production without PRIVATE_NOTE_KEY) every path that writes or opens
// the blob refuses with 503 storage_unavailable. The plaintext is never logged.

// Sealer encrypts the blob at rest (*loss.NoteBox).
type Sealer interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Disabled() bool
}

// ErrStorage: the PII box is unavailable (no key) or a blob does not open — the handlers answer 503.
var ErrStorage = fail(503, "storage_unavailable", "")

// WithSealer wires the PII box (nil → every PII path fails closed). It returns s for chaining.
func (s *Service) WithSealer(b Sealer) *Service {
	s.seal = b
	return s
}

func (s *Service) sealOK() bool { return s.seal != nil && !s.seal.Disabled() }

type addressPII struct {
	Label      string `json:"label,omitempty"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Line       string `json:"line"`
	PostalCode string `json:"postal_code,omitempty"`
}

type orderPII struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	PostalCode string `json:"postal_code,omitempty"`
}

func addressAD(userID uint64) []byte { return []byte("shop-address:" + strconv.FormatUint(userID, 10)) }

func orderAD(userID uint64) []byte { return []byte("shop-order:" + strconv.FormatUint(userID, 10)) }

func (s *Service) sealJSON(v any, ad []byte) (sql.NullString, error) {
	if !s.sealOK() {
		return sql.NullString{}, ErrStorage
	}
	blob, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, ErrStorage
	}
	sealed, err := s.seal.SealWith(string(blob), ad)
	if err != nil {
		return sql.NullString{}, ErrStorage
	}
	return sql.NullString{String: sealed, Valid: true}, nil
}

func (s *Service) openJSON(enc sql.NullString, ad []byte, v any) error {
	if !s.sealOK() {
		return ErrStorage
	}
	plain, err := s.seal.OpenWith(enc.String, ad)
	if err != nil || json.Unmarshal([]byte(plain), v) != nil {
		return ErrStorage
	}
	return nil
}

func hasBlob(enc sql.NullString) bool { return enc.Valid && enc.String != "" }

// OpenAddress fills a's identifying fields from its sealed blob (a legacy row without a blob is returned as stored).
func (s *Service) OpenAddress(a store.ShopAddress) (store.ShopAddress, error) {
	if !hasBlob(a.PiiEnc) {
		return a, nil
	}
	var p addressPII
	if err := s.openJSON(a.PiiEnc, addressAD(a.UserID), &p); err != nil {
		return a, err
	}
	a.Label, a.RecipientName, a.RecipientPhone, a.AddressLine, a.PostalCode = ns(p.Label), p.Name, p.Phone, p.Line, ns(p.PostalCode)
	a.PiiEnc = sql.NullString{}
	return a, nil
}

func (s *Service) openAddresses(list []store.ShopAddress) ([]store.ShopAddress, error) {
	for i, a := range list {
		o, err := s.OpenAddress(a)
		if err != nil {
			return nil, err
		}
		list[i] = o
	}
	return list, nil
}

// OpenOrder fills o's recipient snapshot from its sealed blob (a legacy row without a blob is returned as stored).
func (s *Service) OpenOrder(o store.ShopOrder) (store.ShopOrder, error) {
	if !hasBlob(o.PiiEnc) {
		return o, nil
	}
	var p orderPII
	if err := s.openJSON(o.PiiEnc, orderAD(o.UserID), &p); err != nil {
		return o, err
	}
	o.RecipientName, o.RecipientPhone, o.AddressText, o.PostalCode = p.Name, p.Phone, p.Address, ns(p.PostalCode)
	o.PiiEnc = sql.NullString{}
	return o, nil
}

// OpenRecipientName is the recipient name of an order row read through a projection (the admin shipment queue).
func (s *Service) OpenRecipientName(userID uint64, enc sql.NullString, plain string) (string, error) {
	if !hasBlob(enc) {
		return plain, nil
	}
	var p orderPII
	if err := s.openJSON(enc, orderAD(userID), &p); err != nil {
		return "", err
	}
	return p.Name, nil
}

// IsStorage reports whether err is the fail-closed PII refusal.
func IsStorage(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == "storage_unavailable"
}
