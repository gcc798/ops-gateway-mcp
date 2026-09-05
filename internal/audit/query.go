package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/gcc798/ops-gateway-mcp/internal/pagination"
)

type Filter struct {
	pagination.Query
	From, To                                                            time.Time
	Tool, ResourceType, Resource, Environment, Client, Decision, Status string
}
type Summary struct {
	Total   int `json:"total"`
	Allow   int `json:"allow"`
	Confirm int `json:"confirm"`
	Deny    int `json:"deny"`
	Failed  int `json:"failed"`
}

type FilterOptions struct {
	Tools   []string `json:"tools"`
	Clients []string `json:"clients"`
}

func (s *Store) FilterOptions(ctx context.Context) (FilterOptions, error) {
	out := FilterOptions{Tools: []string{}, Clients: []string{}}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err := tx.SelectContext(ctx, &out.Tools, "SELECT DISTINCT tool FROM operations WHERE tool <> '' ORDER BY tool"); err != nil {
		return out, err
	}
	if err := tx.SelectContext(ctx, &out.Clients, "SELECT DISTINCT client FROM operations WHERE client <> '' ORDER BY client"); err != nil {
		return out, err
	}
	return out, tx.Commit()
}

func (s *Store) Search(ctx context.Context, f Filter) (pagination.Result[Operation], error) {
	out := pagination.Result[Operation]{Items: []Operation{}}
	if err := f.Validate(); err != nil {
		return out, err
	}
	out.Page, out.PageSize = f.Page, f.PageSize
	if !f.From.IsZero() && !f.To.IsZero() && !f.From.Before(f.To) {
		return out, fmt.Errorf("from must precede to")
	}
	where, args := " WHERE 1=1", []any{}
	for _, field := range []struct{ key, value string }{{"tool", f.Tool}, {"resource_type", f.ResourceType}, {"resource", f.Resource}, {"environment", f.Environment}, {"client", f.Client}, {"policy_decision", f.Decision}, {"status", f.Status}} {
		if field.value != "" {
			where += " AND " + field.key + "=?"
			args = append(args, field.value)
		}
	}
	if !f.From.IsZero() {
		where += " AND julianday(timestamp)>=julianday(?)"
		args = append(args, f.From.UTC().Format(time.RFC3339Nano))
	}
	if !f.To.IsZero() {
		where += " AND julianday(timestamp)<julianday(?)"
		args = append(args, f.To.UTC().Format(time.RFC3339Nano))
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM operations`+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	out.Items, err = selectOperations(ctx, tx, operationSelect+where+` ORDER BY julianday(timestamp) DESC,operation_id DESC LIMIT ? OFFSET ?`, append(args, f.PageSize, f.Offset())...)
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
func (s *Store) Summary(ctx context.Context) (Summary, error) {
	var v Summary
	err := s.db.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(policy_decision='allow'),0),coalesce(sum(policy_decision='confirm'),0),coalesce(sum(policy_decision='deny'),0),coalesce(sum(status='failed'),0) FROM operations`).Scan(&v.Total, &v.Allow, &v.Confirm, &v.Deny, &v.Failed)
	return v, err
}
