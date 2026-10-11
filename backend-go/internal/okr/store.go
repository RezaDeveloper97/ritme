package okr

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const objCols = "id, title, owner, period, DATE_FORMAT(starts_on,'%Y-%m-%d'), DATE_FORMAT(ends_on,'%Y-%m-%d'), sort_order"

// scanObj scans objCols (dates arrive as text: the driver is not configured with parseTime).
func scanObj(r interface{ Scan(...any) error }) (Objective, error) {
	var o Objective
	var a, b string
	if err := r.Scan(&o.ID, &o.Title, &o.Owner, &o.Period, &a, &b, &o.SortOrder); err != nil {
		return o, err
	}
	o.StartsOn, o.EndsOn = ParseDay(a), ParseDay(b)
	return o, nil
}

// ParseDay reads a YYYY-MM-DD date (UTC midnight; zero time on a bad value).
func ParseDay(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

const krCols = "id, objective_id, title, metric_key, unit, baseline, target, manual_value, owner, sort_order"

// List returns the objectives (optionally of one period label) with their key results, newest period first.
func (s *Service) List(ctx context.Context, period string) ([]Objective, error) {
	q := "SELECT " + objCols + " FROM okr_objectives"
	var args []any
	if period != "" {
		q += " WHERE period = ?"
		args = append(args, period)
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY starts_on DESC, sort_order, id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objs []Objective
	idx := map[uint64]int{}
	for rows.Next() {
		o, err := scanObj(rows)
		if err != nil {
			return nil, err
		}
		idx[o.ID] = len(objs)
		objs = append(objs, o)
	}
	if err := rows.Err(); err != nil || len(objs) == 0 {
		return objs, err
	}
	kr, err := s.db.QueryContext(ctx, "SELECT "+krCols+" FROM okr_key_results ORDER BY sort_order, id")
	if err != nil {
		return nil, err
	}
	defer kr.Close()
	for kr.Next() {
		k, err := scanKR(kr)
		if err != nil {
			return nil, err
		}
		if i, ok := idx[k.ObjectiveID]; ok {
			objs[i].KeyResults = append(objs[i].KeyResults, k)
		}
	}
	return objs, kr.Err()
}

func scanKR(r interface{ Scan(...any) error }) (KeyResult, error) {
	var k KeyResult
	err := r.Scan(&k.ID, &k.ObjectiveID, &k.Title, &k.MetricKey, &k.Unit, &k.Baseline, &k.Target, &k.ManualValue, &k.Owner, &k.SortOrder)
	return k, err
}

// Periods lists the distinct period labels, newest first.
func (s *Service) Periods(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT period FROM okr_objectives GROUP BY period ORDER BY MAX(starts_on) DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ObjectiveInput is the writable part of an objective.
type ObjectiveInput struct {
	Title, Period    string
	Owner            sql.NullString
	StartsOn, EndsOn time.Time
	SortOrder        int
}

// KRInput is the writable part of a key result.
type KRInput struct {
	Title, Unit      string
	MetricKey, Owner sql.NullString
	Baseline, Target float64
	ManualValue      sql.NullFloat64
	SortOrder        int
}

// CreateObjective inserts an objective.
func (s *Service) CreateObjective(ctx context.Context, in ObjectiveInput, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO okr_objectives (title, owner, period, starts_on, ends_on, sort_order, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
		in.Title, in.Owner, in.Period, in.StartsOn.Format("2006-01-02"), in.EndsOn.Format("2006-01-02"), in.SortOrder, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // AUTO_INCREMENT ids are positive
}

// UpdateObjective rewrites an objective; false when it does not exist.
func (s *Service) UpdateObjective(ctx context.Context, id uint64, in ObjectiveInput, now time.Time) (bool, error) {
	if _, err := s.Objective(ctx, id); err != nil {
		return false, nilIfNoRows(err)
	}
	_, err := s.db.ExecContext(ctx, "UPDATE okr_objectives SET title=?, owner=?, period=?, starts_on=?, ends_on=?, sort_order=?, updated_at=? WHERE id=?",
		in.Title, in.Owner, in.Period, in.StartsOn.Format("2006-01-02"), in.EndsOn.Format("2006-01-02"), in.SortOrder, now, id)
	return err == nil, err
}

func nilIfNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

// Objective loads one objective with its key results.
func (s *Service) Objective(ctx context.Context, id uint64) (Objective, error) {
	return scanObj(s.db.QueryRowContext(ctx, "SELECT "+objCols+" FROM okr_objectives WHERE id = ?", id))
}

// DeleteObjective removes an objective (its key results cascade); false when missing.
func (s *Service) DeleteObjective(ctx context.Context, id uint64) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM okr_objectives WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// KeyResult loads one key result.
func (s *Service) KeyResult(ctx context.Context, id uint64) (KeyResult, error) {
	return scanKR(s.db.QueryRowContext(ctx, "SELECT "+krCols+" FROM okr_key_results WHERE id = ?", id))
}

// CreateKR inserts a key result under an objective.
func (s *Service) CreateKR(ctx context.Context, objectiveID uint64, in KRInput, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO okr_key_results (objective_id, title, metric_key, unit, baseline, target, manual_value, owner, sort_order, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		objectiveID, in.Title, in.MetricKey, in.Unit, in.Baseline, in.Target, in.ManualValue, in.Owner, in.SortOrder, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // AUTO_INCREMENT ids are positive
}

// UpdateKR rewrites a key result.
func (s *Service) UpdateKR(ctx context.Context, id uint64, in KRInput, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE okr_key_results SET title=?, metric_key=?, unit=?, baseline=?, target=?, manual_value=?, owner=?, sort_order=?, updated_at=? WHERE id=?",
		in.Title, in.MetricKey, in.Unit, in.Baseline, in.Target, in.ManualValue, in.Owner, in.SortOrder, now, id)
	return err
}

// DeleteKR removes a key result; false when missing.
func (s *Service) DeleteKR(ctx context.Context, id uint64) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM okr_key_results WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
