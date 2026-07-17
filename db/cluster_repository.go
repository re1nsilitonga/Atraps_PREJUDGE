package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"prime/core/layer1"
)

type ClusterRepository struct {
	pool *pgxpool.Pool
}

func NewClusterRepository(pool *pgxpool.Pool) *ClusterRepository {
	return &ClusterRepository{pool: pool}
}

func (r *ClusterRepository) Upsert(ctx context.Context, c layer1.Cluster) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM fingerprint_clusters WHERE hosting_ip = $1`, c.HostingIP).Scan(&id)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = r.pool.QueryRow(ctx, `
			INSERT INTO fingerprint_clusters
				(registrar, hosting_ip, nameserver, tld, domain_count,
				 first_registration_date, last_registration_date, registration_window_hours, registration_burst_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, nullableString(c.Registrar), c.HostingIP, nullableString(c.Nameserver), nullableString(c.TLD), len(c.Domains),
			c.FirstRegistrationDate, c.LastRegistrationDate, nullableInt(c.RegistrationWindowHours), c.RegistrationBurstScore).
			Scan(&id)
	case err != nil:
		return "", err
	default:
		_, err = r.pool.Exec(ctx, `
			UPDATE fingerprint_clusters SET
				domain_count = $2, first_registration_date = $3, last_registration_date = $4,
				registration_window_hours = $5, registration_burst_score = $6
			WHERE id = $1
		`, id, len(c.Domains), c.FirstRegistrationDate, c.LastRegistrationDate, nullableInt(c.RegistrationWindowHours), c.RegistrationBurstScore)
	}
	if err != nil {
		return "", err
	}

	_, err = r.pool.Exec(ctx, `UPDATE domains SET cluster_id = $1 WHERE domain = ANY($2)`, id, c.Domains)
	return id, err
}

func (r *ClusterRepository) ListClusters(ctx context.Context) ([]layer1.Cluster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, host(hosting_ip), COALESCE(nameserver, ''), COALESCE(registrar, ''), COALESCE(tld, ''), registration_burst_score
		FROM fingerprint_clusters
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []layer1.Cluster
	for rows.Next() {
		var c layer1.Cluster
		if err := rows.Scan(&c.ID, &c.HostingIP, &c.Nameserver, &c.Registrar, &c.TLD, &c.RegistrationBurstScore); err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableInt(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}
