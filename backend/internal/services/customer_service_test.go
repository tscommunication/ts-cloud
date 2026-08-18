package services

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateCustomerWriteError(t *testing.T) {
	originalErr := errors.New("database unavailable")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "nil error",
			err:  nil,
			want: nil,
		},
		{
			name: "duplicate mobile",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "idx_customers_mobile_unique",
			},
			want: ErrCustomerMobileExists,
		},
		{
			name: "duplicate NID",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "idx_customers_nid_unique",
			},
			want: ErrCustomerNIDExists,
		},
		{
			name: "unknown unique constraint",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "some_other_unique_constraint",
			},
		},
		{
			name: "non unique postgres error",
			err: &pgconn.PgError{
				Code:           "23503",
				ConstraintName: "some_foreign_key",
			},
		},
		{
			name: "ordinary error",
			err:  originalErr,
			want: originalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateCustomerWriteError(tt.err)

			if tt.want != nil {
				if !errors.Is(got, tt.want) {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
				return
			}

			if got != tt.err {
				t.Fatalf("expected original error to be preserved")
			}
		})
	}
}
