package redis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureClient(t *testing.T) *Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := Connect(ctx, Settings{Host: "127.0.0.1", Port: 6380})
	require.NoError(t, err, "start the required fixture with docker compose up -d redis")
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	ready, err := client.Execute(ctx, 0, "GET fixture:seed:ready")
	require.NoError(t, err)
	require.Equal(t, "1", ready.Raw, "the Redis Compose fixture must finish seeding")
	return client
}

func TestConnectionOptions(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		address  string
		host     string
		username string
		password string
		database int
		wantErr  string
	}{
		{name: "host and port", settings: Settings{Host: " localhost ", Port: 6380, Username: " reader ", Password: "secret"}, address: "localhost:6380", host: "localhost", username: "reader", password: "secret"},
		{name: "IPv6 host", settings: Settings{Host: "::1", Port: 6380}, address: "[::1]:6380", host: "::1"},
		{name: "DSN takes precedence", settings: Settings{Host: "ignored", Port: -1, DSN: "redis://alice:secret@example.test:6390/0"}, address: "example.test:6390", host: "example.test", username: "alice", password: "secret"},
		{name: "missing host", settings: Settings{Port: 6379}, wantErr: "host is required"},
		{name: "invalid port", settings: Settings{Host: "localhost", Port: 0}, wantErr: "port must be"},
		{name: "invalid DSN scheme", settings: Settings{DSN: "http://example.test"}, wantErr: "valid redis:// DSN"},
		{name: "DSN selects db1", settings: Settings{DSN: "redis://example.test/1"}, address: "example.test:6379", host: "example.test", database: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, host, err := connectionOptions(test.settings)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.address, options.Addr)
			assert.Equal(t, test.host, host)
			assert.Equal(t, test.username, options.Username)
			assert.Equal(t, test.password, options.Password)
			assert.Equal(t, test.database, options.DB)
		})
	}
}

func TestConnectDiscoversDatabasesAndClosesClients(t *testing.T) {
	client := fixtureClient(t)
	assert.Equal(t, "127.0.0.1", client.Host())
	assert.GreaterOrEqual(t, client.DatabaseCount(), 2)
	base, err := client.clientFor(0)
	require.NoError(t, err)
	other, err := client.clientFor(1)
	require.NoError(t, err)
	assert.NotSame(t, base, other)
	again, err := client.clientFor(1)
	require.NoError(t, err)
	assert.Same(t, other, again)
	_, err = client.clientFor(-1)
	assert.ErrorContains(t, err, "out of range")
	_, err = client.clientFor(client.DatabaseCount())
	assert.ErrorContains(t, err, "out of range")
	require.NoError(t, client.Close())
	assert.NoError(t, client.Close())
	_, err = client.clientFor(0)
	assert.ErrorContains(t, err, "closed")
}

func TestPopulatedDatabasesIncludesFixtureDatabases(t *testing.T) {
	client := fixtureClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	databases, err := client.PopulatedDatabases(ctx)
	require.NoError(t, err)
	assert.True(t, slices.IsSorted(databases))
	assert.Contains(t, databases, 0)
	assert.Contains(t, databases, 1)
}

func TestPopulatedDatabasesReusesDSNClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := Connect(ctx, Settings{DSN: "redis://127.0.0.1:6380/1"})
	require.NoError(t, err, "start the required fixture with docker compose up -d redis")
	t.Cleanup(func() { assert.NoError(t, client.Close()) })

	databases, err := client.PopulatedDatabases(ctx)
	require.NoError(t, err)
	assert.Contains(t, databases, 0)
	assert.Contains(t, databases, 1)
	assert.Len(t, client.clients, 1)
}

func TestConnectFailsWhenServerIsUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().(*net.TCPAddr)
	require.NoError(t, listener.Close())
	for _, test := range []struct {
		name     string
		settings Settings
	}{
		{name: "host and port", settings: Settings{Host: "127.0.0.1", Port: address.Port}},
		{name: "DSN", settings: Settings{DSN: "redis://" + address.String()}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			client, err := Connect(ctx, test.settings)
			assert.Nil(t, client)
			assert.ErrorContains(t, err, "connect to Redis")
		})
	}
}

func TestConnectFailureDoesNotWriteToTerminal(t *testing.T) {
	// Use a subprocess because go-redis captures stderr during initialization.
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestConnectFailsWhenServerIsUnavailable$")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.Output()
	require.NoError(t, err, "stdout: %s; stderr: %s", stdout, stderr.String())
	assert.Empty(t, stderr.String(), "Redis diagnostics must not bypass the connection modal")
}

func TestConfiguredDatabaseCount(t *testing.T) {
	count, err := configuredDatabaseCount(map[string]string{"databases": "16"})
	require.NoError(t, err)
	assert.Equal(t, 16, count)
	for _, value := range []string{"", "abc", "0", "-1"} {
		_, err := configuredDatabaseCount(map[string]string{"databases": value})
		assert.ErrorContains(t, err, "invalid logical database count")
	}
}

func TestParsePopulatedDatabases(t *testing.T) {
	info := "# Keyspace\r\ndb3:keys=2,expires=0\r\ndb0:keys=1,expires=0\r\ndb1:keys=0,expires=0\r\ndb99:keys=1,expires=0\r\n"
	databases, err := parsePopulatedDatabases(info, 16)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 3}, databases)
	_, err = parsePopulatedDatabases("db1:keys=invalid,expires=0", 16)
	assert.ErrorContains(t, err, "invalid Redis key count")
}

func TestLoadKeyPageTraversesFixtureWithoutDuplicates(t *testing.T) {
	client := fixtureClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	page, err := client.LoadKeyPage(ctx, 0, 400, ScanCursor{})
	require.NoError(t, err)
	require.Len(t, page.Keys, 400)
	require.True(t, page.HasMore)

	seen := make(map[string]struct{})
	pages := 0
	for {
		pages++
		for _, key := range page.Keys {
			_, duplicate := seen[key.Name]
			assert.False(t, duplicate, "duplicate key %q", key.Name)
			seen[key.Name] = struct{}{}
		}
		if !page.HasMore {
			break
		}
		page, err = client.LoadKeyPage(ctx, 0, 400, page.Next)
		require.NoError(t, err)
		require.NotEmpty(t, page.Keys)
	}
	assert.Greater(t, pages, 1)
	assert.Contains(t, seen, "fixture:app:name")
}

func TestLoadKeyPagePreservesPendingNamesAndLookahead(t *testing.T) {
	client := fixtureClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor := ScanCursor{pending: []string{
		"fixture:app:name", "fixture:app:name", "fixture:orders:next_id", "fixture:seed:ready",
	}, done: true}
	first, err := client.LoadKeyPage(ctx, 0, 2, cursor)
	require.NoError(t, err)
	require.Len(t, first.Keys, 2)
	assert.Equal(t, []string{"fixture:app:name", "fixture:orders:next_id"}, []string{first.Keys[0].Name, first.Keys[1].Name})
	assert.True(t, first.HasMore)
	assert.Len(t, cursor.pending, 4, "loading must not mutate the input cursor")
	second, err := client.LoadKeyPage(ctx, 0, 2, first.Next)
	require.NoError(t, err)
	require.Len(t, second.Keys, 1)
	assert.Equal(t, "fixture:seed:ready", second.Keys[0].Name)
	assert.False(t, second.HasMore)
}

func TestLoadKeyPageRejectsBoundsAndCancellation(t *testing.T) {
	client := fixtureClient(t)
	ctx := context.Background()
	_, err := client.LoadKeyPage(ctx, -1, 10, ScanCursor{})
	assert.ErrorContains(t, err, "out of range")
	_, err = client.LoadKeyPage(ctx, client.DatabaseCount(), 10, ScanCursor{})
	assert.ErrorContains(t, err, "out of range")
	_, err = client.LoadKeyPage(ctx, 0, 0, ScanCursor{})
	assert.ErrorContains(t, err, "must be positive")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = client.LoadKeyPage(canceled, 0, 10, ScanCursor{})
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestLoadKeyReadsFixtureTypes(t *testing.T) {
	client := fixtureClient(t)
	connection, err := client.clientFor(0)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tests := []struct {
		name     string
		kind     string
		contains string
	}{
		{name: "fixture:app:name", kind: "string", contains: "db-tui"},
		{name: "fixture:customer:1", kind: "hash", contains: `"name":"Customer 1"`},
		{name: "fixture:recent:queries", kind: "list", contains: "GET fixture:app:name"},
		{name: "fixture:roles:reader", kind: "set", contains: `"1"`},
		{name: "fixture:leaderboard", kind: "zset", contains: `"member":"customer:1"`},
		{name: "fixture:events", kind: "stream", contains: "order_created"},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			key, exists, err := loadKey(ctx, connection, test.name)
			require.NoError(t, err)
			require.True(t, exists)
			assert.Equal(t, test.name, key.Name)
			assert.Equal(t, test.kind, key.Type)
			assert.Contains(t, key.Value, test.contains)
			assert.Equal(t, "—", key.TTL)
		})
	}
	_, exists, err := loadKey(ctx, connection, "test:db-tui:missing-key")
	require.NoError(t, err)
	assert.False(t, exists)
	_, err = readValue(ctx, connection, "fixture:app:name", "unsupported")
	assert.ErrorContains(t, err, "unsupported Redis key type")
}

func TestFormatTTLAndSafeText(t *testing.T) {
	assert.Equal(t, "—", formatTTL(-1))
	assert.Equal(t, "1.5s", formatTTL(1500*time.Millisecond))
	assert.Equal(t, `"a\nb"`, safeText("a\nb"))
	assert.Equal(t, "0xff", safeText(string([]byte{0xff})))
	assert.Equal(t, "plain", safeText("plain"))
	assert.False(t, strings.Contains(safeText("a\x1bb"), "\x1b"))
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
		wantErr string
	}{
		{name: "plain", command: "GET fixture:app:name", want: []string{"GET", "fixture:app:name"}},
		{name: "quoted arguments", command: `SET "a b" 'c d'`, want: []string{"SET", "a b", "c d"}},
		{name: "escaped space", command: `SET key hello\ world`, want: []string{"SET", "key", "hello world"}},
		{name: "empty quoted value", command: `SET key ""`, want: []string{"SET", "key", ""}},
		{name: "empty", command: "   ", wantErr: "empty"},
		{name: "unmatched quote", command: `GET "unfinished`, wantErr: "unfinished"},
		{name: "unfinished escape", command: "GET key\\", wantErr: "unfinished"},
		{name: "second command", command: "PING\nGET key", wantErr: "one Redis command"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arguments, err := parseCommand(test.command)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, arguments)
		})
	}
}

func TestRawReplyFormatsRedisTypes(t *testing.T) {
	tests := []struct {
		name  string
		reply any
		want  string
	}{
		{name: "RESP3 map", reply: map[any]any{"name": "Customer 1", "active": "true"}, want: "active: true\nname: Customer 1"},
		{name: "string map", reply: map[string]string{"b": "2", "a": "1"}, want: "a: 1\nb: 2"},
		{name: "interface map", reply: map[string]any{"score": int64(7)}, want: "score: 7"},
		{name: "array", reply: []any{"one", int64(2), nil}, want: "one\n2\n(nil)"},
		{name: "empty array", reply: []any{}, want: "(empty)"},
		{name: "empty map", reply: map[any]any{}, want: "(empty)"},
		{name: "nil", reply: nil, want: "(nil)"},
		{name: "scalar", reply: int64(42), want: "42"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, rawReply(test.reply))
			assert.NotContains(t, rawReply(test.reply), "map[")
		})
	}
}

func TestRawReplyPreservesNewlinesAndRemovesControls(t *testing.T) {
	assert.Equal(t, "line 1\nline 2�[31m", rawReply("line 1\r\nline 2\x1b[31m"))
	assert.Equal(t, "0xff", rawReply([]byte{0xff}))
}

func TestExecuteFixtureCommands(t *testing.T) {
	client := fixtureClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ping, err := client.Execute(ctx, 0, "PING")
	require.NoError(t, err)
	assert.Equal(t, "PONG", ping.Raw)
	value, err := client.Execute(ctx, 0, "GET fixture:app:name")
	require.NoError(t, err)
	assert.Equal(t, "db-tui", value.Raw)
	hash, err := client.Execute(ctx, 0, "HGETALL fixture:customer:1")
	require.NoError(t, err)
	assert.Contains(t, hash.Raw, "name: Customer 1")
	assert.NotContains(t, hash.Raw, "map[")

	selected, err := client.Execute(ctx, 0, "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "OK", selected.Raw)
	require.NotNil(t, selected.SelectDatabase)
	assert.Equal(t, 1, *selected.SelectDatabase)
	db1, err := client.Execute(ctx, 1, "GET fixture:db1:message")
	require.NoError(t, err)
	assert.Equal(t, "hello from db1", db1.Raw)
	missing, err := client.Execute(ctx, 0, "GET test:db-tui:missing-key")
	require.NoError(t, err)
	assert.Equal(t, "(nil)", missing.Raw)

	name := fmt.Sprintf("test:db-tui:command:%d", time.Now().UnixNano())
	connection, err := client.clientFor(0)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = connection.Del(context.Background(), name).Result() })
	set, err := client.Execute(ctx, 0, "SET "+name+" hello")
	require.NoError(t, err)
	assert.Equal(t, "OK", set.Raw)
	stored, err := client.Execute(ctx, 0, "GET "+name)
	require.NoError(t, err)
	assert.Equal(t, "hello", stored.Raw)
	deleted, err := client.Execute(ctx, 0, "DEL "+name)
	require.NoError(t, err)
	assert.Equal(t, "1", deleted.Raw)
}

func TestExecuteRejectsInvalidCommands(t *testing.T) {
	client := fixtureClient(t)
	ctx := context.Background()
	_, err := client.Execute(ctx, -2, "PING")
	assert.ErrorContains(t, err, "out of range")
	_, err = client.Execute(ctx, 0, "SELECT 999")
	assert.ErrorContains(t, err, "out of range")
	_, err = client.Execute(ctx, 0, "SELECT")
	assert.ErrorContains(t, err, "one database index")
	_, err = client.Execute(ctx, 0, "PING\nPING")
	assert.ErrorContains(t, err, "one Redis command")
	_, err = client.Execute(ctx, 0, "DB_TUI_NO_SUCH_COMMAND")
	assert.ErrorContains(t, err, "execute Redis command")
}
