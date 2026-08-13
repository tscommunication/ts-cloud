package linux

import "testing"

func TestParseVSFTPDLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected *VSFTPDEvent
	}{
		{
			name: "successful login",
			line: `Thu Aug 13 10:00:00 2026 [pid 101] [alice] OK LOGIN: Client "192.0.2.10"`,
			expected: &VSFTPDEvent{
				Type: EventLoginSuccess, Username: "alice", IP: "192.0.2.10",
			},
		},
		{
			name: "failed login",
			line: `Thu Aug 13 10:01:00 2026 [pid 102] [bob] FAIL LOGIN: Client "192.0.2.11"`,
			expected: &VSFTPDEvent{
				Type: EventLoginFailed, Username: "bob", IP: "192.0.2.11",
			},
		},
		{
			name: "upload",
			line: `Thu Aug 13 10:02:00 2026 [pid 103] [carol] OK UPLOAD: Client "192.0.2.12", "/movie.mkv", 4096 bytes`,
			expected: &VSFTPDEvent{
				Type: EventUpload, Username: "carol", IP: "192.0.2.12",
				FileName: "/movie.mkv", Bytes: 4096,
			},
		},
		{
			name: "download",
			line: `Thu Aug 13 10:03:00 2026 [pid 104] [dave] OK DOWNLOAD: Client "192.0.2.13", "/report.pdf", 2048 bytes`,
			expected: &VSFTPDEvent{
				Type: EventDownload, Username: "dave", IP: "192.0.2.13",
				FileName: "/report.pdf", Bytes: 2048,
			},
		},
		{
			name: "unknown line",
			line: "unrelated log entry",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := ParseVSFTPDLine(test.line)
			if test.expected == nil {
				if actual != nil {
					t.Fatalf("expected nil event, got %+v", actual)
				}
				return
			}
			if actual == nil {
				t.Fatal("expected event, got nil")
			}
			if *actual != *test.expected {
				t.Fatalf("unexpected event: got %+v, want %+v", actual, test.expected)
			}
		})
	}
}
