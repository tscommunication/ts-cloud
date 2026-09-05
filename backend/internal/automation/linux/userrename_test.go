package linux

import (
	"errors"
	"reflect"
	"testing"
)

func TestRenameUserBuildsSafeUsermodCommand(t *testing.T) {
	oldExecute := renameUserExecute
	t.Cleanup(func() {
		renameUserExecute = oldExecute
	})

	var gotCommand string
	var gotArgs []string

	renameUserExecute = func(
		command string,
		args ...string,
	) (*CommandResult, error) {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		return &CommandResult{}, nil
	}

	// Use a destination name that should not exist on the test host.
	err := RenameUser(
		"tscloud-ftp-old-test-user",
		"tscloud-ftp-new-test-user",
		"/data/ftp/tscloud-ftp-old-test-user",
		"/data/ftp/tscloud-ftp-new-test-user",
	)
	if err != nil {
		t.Fatal(err)
	}

	if gotCommand != "sudo" {
		t.Fatalf("command = %q, want sudo", gotCommand)
	}

	wantArgs := []string{
		"/usr/sbin/usermod",
		"-l",
		"tscloud-ftp-new-test-user",
		"-d",
		"/data/ftp/tscloud-ftp-new-test-user",
		"-m",
		"tscloud-ftp-old-test-user",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			wantArgs,
		)
	}
}

func TestRenameUserRejectsUnsafeHomeMapping(t *testing.T) {
	oldExecute := renameUserExecute
	t.Cleanup(func() {
		renameUserExecute = oldExecute
	})

	called := false
	renameUserExecute = func(
		command string,
		args ...string,
	) (*CommandResult, error) {
		called = true
		return &CommandResult{}, nil
	}

	tests := []struct {
		name    string
		oldUser string
		newUser string
		oldHome string
		newHome string
	}{
		{
			name:    "different root",
			oldUser: "old-user",
			newUser: "new-user",
			oldHome: "/data/ftp/old-user",
			newHome: "/other/root/new-user",
		},
		{
			name:    "old home mismatch",
			oldUser: "old-user",
			newUser: "new-user",
			oldHome: "/data/ftp/not-old-user",
			newHome: "/data/ftp/new-user",
		},
		{
			name:    "new home mismatch",
			oldUser: "old-user",
			newUser: "new-user",
			oldHome: "/data/ftp/old-user",
			newHome: "/data/ftp/not-new-user",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := RenameUser(
				test.oldUser,
				test.newUser,
				test.oldHome,
				test.newHome,
			)
			if err == nil {
				t.Fatal("expected unsafe rename to fail")
			}
		})
	}

	if called {
		t.Fatal("unsafe rename executed usermod")
	}
}

func TestRenameUserPropagatesCommandFailure(t *testing.T) {
	oldExecute := renameUserExecute
	t.Cleanup(func() {
		renameUserExecute = oldExecute
	})

	renameUserExecute = func(
		command string,
		args ...string,
	) (*CommandResult, error) {
		return &CommandResult{},
			errors.New("simulated usermod failure")
	}

	err := RenameUser(
		"tscloud-ftp-old-fail-user",
		"tscloud-ftp-new-fail-user",
		"/data/ftp/tscloud-ftp-old-fail-user",
		"/data/ftp/tscloud-ftp-new-fail-user",
	)

	if err == nil {
		t.Fatal("expected command failure")
	}
}
