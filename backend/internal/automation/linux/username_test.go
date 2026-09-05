package linux

import "testing"

func TestValidateManagedUsernameAcceptsCurrentPPPoEShapes(
	t *testing.T,
) {
	valid := []string{
		"testuser01",
		"saiful",
		"Par_002_morad",
		"Fw_000_saiful",
		"Shukur_001_control",
		"F_008_krishna",
		"Wk_009_shohagh-counter",
		"Ramnagor_015_sajib",
		"Belnagor_020_alamin",
		"Par_035_RHD-2",
		"tscloud-write-test-001",
	}

	for _, username := range valid {
		t.Run(username, func(t *testing.T) {
			if err := ValidateManagedUsername(username); err != nil {
				t.Fatalf(
					"expected %q to be accepted: %v",
					username,
					err,
				)
			}
		})
	}
}

func TestValidateManagedUsernameRejectsUnsafeNames(
	t *testing.T,
) {
	invalid := []string{
		"",
		" ",
		"../escape",
		"bad/name",
		"bad user",
		"bad:name",
		"bad\nname",
		"-leading-hyphen",
		".leading-dot",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567",
	}

	for _, username := range invalid {
		t.Run(username, func(t *testing.T) {
			if err := ValidateManagedUsername(username); err == nil {
				t.Fatalf(
					"expected %q to be rejected",
					username,
				)
			}
		})
	}
}
