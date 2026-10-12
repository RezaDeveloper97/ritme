package accountexport

import (
	"context"
	"database/sql"

	joinstore "github.com/ritme/backend-go/internal/cityservices/join/store"
	"github.com/ritme/backend-go/internal/insurance"
	insstore "github.com/ritme/backend-go/internal/insurance/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/shop/order"
	shopstore "github.com/ritme/backend-go/internal/shop/store"
)

// insuranceData: {policies, members, coverages, claims, claim_events, claim_documents, questionnaires} (CB-INS). The
// claim and step notes, the questionnaire explanation and the evidence copied from her record are opened for her
// (null when the seal does not open); claim documents are links to her record documents (ids and labels only).
func insuranceData(db *sql.DB, svc *insurance.Service) loader {
	q := insstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "policies", load: func() ([]any, error) {
				l, err := q.ExportInsurancePolicies(ctx, uid)
				return rows("insurance policies", l, err, nil, nil, nil)
			}},
			part{key: "members", load: func() ([]any, error) {
				l, err := q.ExportInsurancePolicyMembers(ctx, uid)
				return rows("insurance members", l, err, nil, nil, nil)
			}},
			part{key: "coverages", load: func() ([]any, error) {
				l, err := q.ExportInsuranceCoverages(ctx, uid)
				return rows("insurance coverages", l, err, nil, nil, nil)
			}},
			part{key: "claims", load: func() ([]any, error) {
				l, err := q.ExportInsuranceClaims(ctx, uid)
				return rows("insurance claims", l, err, nil, []string{"notes_enc"}, func(r *insstore.InsuranceClaim, o *jsonx.OrderedMap) {
					o.Set("note", nonEmpty(svc.ClaimNote(*r)))
				})
			}},
			part{key: "claim_events", load: func() ([]any, error) {
				l, err := q.ExportInsuranceClaimEvents(ctx, uid)
				return rows("insurance claim events", l, err, nil, []string{"note_enc"}, func(r *insstore.InsuranceClaimEvent, o *jsonx.OrderedMap) {
					o.Set("note", nonEmpty(svc.EventNote(uid, *r)))
				})
			}},
			part{key: "claim_documents", load: func() ([]any, error) {
				l, err := q.ExportInsuranceClaimDocuments(ctx, uid)
				return rows("insurance claim documents", l, err, nil, nil, nil)
			}},
			part{key: "questionnaires", load: func() ([]any, error) {
				l, err := q.ExportInsuranceQuestionnaires(ctx, uid)
				return rows("insurance questionnaires", l, err, model.Casts{"timeline": model.CastArray, "confirmed_at": model.CastDateTime},
					[]string{"evidence_enc", "explanation_enc"}, func(r *insstore.InsuranceQuestionnaire, o *jsonx.OrderedMap) {
						answers := []any{}
						for _, a := range svc.Answers(*r) {
							evidence := a.Evidence
							if evidence == nil {
								evidence = []string{}
							}
							answers = append(answers, jsonx.Obj("code", a.Code, "answer", strPtr(a.Answer),
								"record_answer", strPtr(a.RecordAnswer), "evidence", evidence))
						}
						o.Set("answers", answers)
						o.Set("explanation", nonEmpty(svc.Explanation(*r)))
					})
			}},
		)
	}
}

// shopData: {addresses, orders, shipments, order_items, order_events, returns, reviews, checklist, supply_reminders}
// (CB-SHOP). Saved addresses and the order recipient snapshots are opened for her (order.OpenAddress / OpenOrder); a
// blob that does not open keeps the stored (empty) columns. Internal keys (idempotency key) are left out.
func shopData(db *sql.DB, svc *order.Service) loader {
	q := shopstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "addresses", load: func() ([]any, error) {
				l, err := q.ExportShopAddresses(ctx, uid)
				for i := range l {
					if a, oerr := svc.OpenAddress(l[i]); oerr == nil {
						l[i] = a
					}
				}
				return rows("shop addresses", l, err, model.Casts{"is_default": model.CastBoolean}, []string{"pii_enc"}, nil)
			}},
			part{key: "orders", load: func() ([]any, error) {
				l, err := q.ExportShopOrders(ctx, uid)
				for i := range l {
					if o, oerr := svc.OpenOrder(l[i]); oerr == nil {
						l[i] = o
					}
				}
				return rows("shop orders", l, err, model.Casts{"discreet": model.CastBoolean, "placed_at": model.CastDateTime,
					"cancelled_at": model.CastDateTime}, []string{"pii_enc", "idempotency_key"}, nil)
			}},
			part{key: "shipments", load: func() ([]any, error) {
				l, err := q.ExportShopOrderShipments(ctx, uid)
				return rows("shop shipments", l, err, model.Casts{"delivered_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "order_items", load: func() ([]any, error) {
				l, err := q.ExportShopOrderItems(ctx, uid)
				return rows("shop order items", l, err, nil, nil, func(_ *shopstore.ShopOrderItem, o *jsonx.OrderedMap) {
					jsonText(o, "title", "variant_label")
				})
			}},
			part{key: "order_events", load: func() ([]any, error) {
				l, err := q.ExportShopOrderEvents(ctx, uid)
				return rows("shop order events", l, err, nil, nil, nil)
			}},
			part{key: "returns", load: func() ([]any, error) {
				l, err := q.ExportShopReturns(ctx, uid)
				return rows("shop returns", l, err, nil, nil, nil)
			}},
			part{key: "reviews", load: func() ([]any, error) {
				l, err := q.ExportShopProductReviews(ctx, uid)
				return rows("shop reviews", l, err, nil, nil, func(_ *shopstore.ShopProductReview, o *jsonx.OrderedMap) {
					jsonText(o, "variant_label")
				})
			}},
			part{key: "checklist", load: func() ([]any, error) {
				l, err := q.ExportShopChecklistItems(ctx, uid)
				return rows("shop checklist", l, err, nil, nil, nil)
			}},
			part{key: "supply_reminder", one: true, load: func() ([]any, error) {
				l, err := q.ExportShopSupplyReminders(ctx, uid)
				return rows("shop supply reminder", l, err, model.Casts{"is_enabled": model.CastBoolean}, nil, nil)
			}},
		)
	}
}

// directory: {bookings, reviews, reports, applications, application_files} (CB-DIR). Applications keep their
// document / photo references (file ids); application_files lists those files' metadata only — never the bytes, the
// storage path or the hash.
func directory(db *sql.DB) loader {
	q := joinstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		owner := sql.NullInt64{Int64: int64(uid), Valid: true} //nolint:gosec // G115: user ids fit int64
		return object(
			part{key: "bookings", load: func() ([]any, error) {
				l, err := q.ExportDirectoryBookings(ctx, uid)
				return rows("directory bookings", l, err, model.Casts{"confirmed_at": model.CastDateTime,
					"cancelled_at": model.CastDateTime, "completed_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "reviews", load: func() ([]any, error) {
				l, err := q.ExportDirectoryReviews(ctx, uid)
				return rows("directory reviews", l, err, nil, nil, nil)
			}},
			part{key: "reports", load: func() ([]any, error) {
				l, err := q.ExportDirectoryReports(ctx, owner)
				return rows("directory reports", l, err, model.Casts{"resolved_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "applications", load: func() ([]any, error) {
				l, err := q.ExportDirectoryApplications(ctx, uid)
				return rows("directory applications", l, err, model.Casts{"owner_confirmed": model.CastBoolean,
					"terms_accepted_at": model.CastDateTime, "submitted_at": model.CastDateTime, "reviewed_at": model.CastDateTime},
					nil, func(_ *joinstore.CityServiceApplication, o *jsonx.OrderedMap) {
						jsonText(o, "category_slugs", "age_ranges", "amenities", "weekly_hours", "photo_file_ids", "services", "documents")
					})
			}},
			part{key: "application_files", load: func() ([]any, error) {
				l, err := q.ExportDirectoryApplicationFiles(ctx, owner)
				return rows("directory application files", l, err, nil, nil, nil)
			}},
		)
	}
}

func strPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
