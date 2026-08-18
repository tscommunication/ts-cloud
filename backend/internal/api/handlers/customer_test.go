package handlers

import (
	"errors"
	"net/http"
	"testing"

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
