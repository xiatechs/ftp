package ftp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBogusDataIP(t *testing.T) {
	for _, tC := range []struct {
		cmd, data net.IP
		bogus     bool
	}{
		{net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 1, 1), false},
		{net.IPv4(192, 168, 1, 1), net.IPv4(1, 1, 1, 1), true},
		{net.IPv4(10, 65, 1, 1), net.IPv4(1, 1, 1, 1), true},
		{net.IPv4(10, 65, 25, 1), net.IPv4(10, 65, 8, 1), false},
	} {
		if got, want := isBogusDataIP(tC.cmd, tC.data), tC.bogus; got != want {
			t.Errorf("%s,%s got %t, wanted %t", tC.cmd, tC.data, got, want)
		}
	}
}

func TestEPSV_Parse_Valid(t *testing.T) {
	port, err := parseEPSV("Entering Extended Passive Mode (|||4242|)")
	assert.NoError(t, err)
	assert.Equal(t, 4242, port)
}

func TestEPSV_Parse_MissingTrailingPipe_ShouldError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("epsv() panicked, want graceful error: %v", r)
		}
	}()
	_, err := parseEPSV("Entering Extended Passive Mode (|||4242)")
	if err == nil {
		t.Fatalf("expected error for malformed EPSV response, got nil")
	}
}

func TestEPSV_Parse_MissingPortBetweenPipes_ShouldError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("epsv() panicked, want graceful error: %v", r)
		}
	}()
	_, err := parseEPSV("Entering Extended Passive Mode (||||)")
	if err == nil {
		t.Fatalf("expected error for malformed EPSV response, got nil")
	}
}

func TestRawCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		closeBefore bool // simulate a connection error before sending the command
		wantCode    int
		wantMsg     string
		wantErr     bool
	}{
		{
			name:     "success",
			command:  "NOOP",
			wantCode: StatusCommandOK,
			wantMsg:  "NOOP ok.",
		},
		{
			name:     "non-2xx response",
			command:  "BOGUS",
			wantCode: StatusBadCommand,
			wantMsg:  "Unknown command BOGUS.",
			wantErr:  true,
		},
		{
			name:        "connection error",
			command:     "NOOP",
			closeBefore: true,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, c := openConn(t, "127.0.0.1")

			if tc.closeBefore {
				if err := c.Quit(); err != nil {
					t.Fatalf("closing connection ahead of test: %s", err)
				}
				mock.Wait()
			}

			code, msg, err := c.RawCommand(tc.command)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantMsg, msg)

			if !tc.closeBefore {
				closeConn(t, mock, c, []string{tc.command})
			}
		})
	}
}
