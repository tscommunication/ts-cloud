package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func TestCustomerDuplicateErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
		wantHandled bool
	}{
		{
			name:        "duplicate mobile",
			err:         services.ErrCustomerMobileExists,
			wantStatus:  http.StatusConflict,
			wantMessage: "A customer with this mobile number already exists",
			wantHandled: true,
		},
		{
			name:        "wrapped duplicate mobile",
			err:         errors.Join(errors.New("write failed"), services.ErrCustomerMobileExists),
			wantStatus:  http.StatusConflict,
			wantMessage: "A customer with this mobile number already exists",
			wantHandled: true,
		},
		{
			name:        "duplicate NID",
			err:         services.ErrCustomerNIDExists,
			wantStatus:  http.StatusConflict,
			wantMessage: "A customer with this NID already exists",
			wantHandled: true,
		},
		{
			name:        "ordinary error",
			err:         errors.New("database unavailable"),
			wantStatus:  0,
			wantMessage: "",
			wantHandled: false,
		},
		{
			name:        "nil error",
			err:         nil,
			wantStatus:  0,
			wantMessage: "",
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotMessage, gotHandled :=
				customerDuplicateErrorResponse(tt.err)

			if gotStatus != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					gotStatus,
					tt.wantStatus,
				)
			}

			if gotMessage != tt.wantMessage {
				t.Fatalf(
					"message = %q, want %q",
					gotMessage,
					tt.wantMessage,
				)
			}

			if gotHandled != tt.wantHandled {
				t.Fatalf(
					"handled = %v, want %v",
					gotHandled,
					tt.wantHandled,
				)
			}
		})
	}
}

func TestParseOptionalCustomerDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantDay   int
		wantMonth time.Month
		wantYear  int
		wantErr   bool
	}{
		{
			name:    "blank date",
			input:   "",
			wantNil: true,
		},
		{
			name:      "valid DD-MM-YYYY",
			input:     "19-08-2026",
			wantDay:   19,
			wantMonth: time.August,
			wantYear:  2026,
		},
		{
			name:    "reject ISO format",
			input:   "2026-08-19",
			wantErr: true,
		},
		{
			name:    "reject slash format",
			input:   "19/08/2026",
			wantErr: true,
		},
		{
			name:    "reject impossible date",
			input:   "31-02-2026",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptionalCustomerDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected date parsing error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil date, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected parsed date")
			}

			if got.Day() != tt.wantDay ||
				got.Month() != tt.wantMonth ||
				got.Year() != tt.wantYear {
				t.Fatalf(
					"parsed date = %02d-%02d-%04d",
					got.Day(),
					got.Month(),
					got.Year(),
				)
			}
		})
	}
}
