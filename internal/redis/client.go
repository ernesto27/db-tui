// Package redis owns the concrete standalone Redis client used by the TUI.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	goredis "github.com/redis/go-redis/v9"
)

// Settings identifies a standalone Redis server.
type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	DSN      string
}

// Key is one Redis key rendered in the data grid.
type Key struct {
	Name  string
	Value string
	Type  string
	TTL   string
}

// ScanCursor identifies the next page of a logical database key scan.
// Its fields are private so callers cannot change the scan position.
type ScanCursor struct {
	scan    uint64
	pending []string
	visited map[string]struct{}
	ready   *Key
	done    bool
}

// KeyPage holds one page of keys and the cursor for the following page.
type KeyPage struct {
	Keys    []Key
	Next    ScanCursor
	HasMore bool
}

// Result is a finite Redis command reply for the query panel.
type Result struct {
	Raw            string
	CommandTag     string
	SelectDatabase *int
}

// Client owns a pool for each logical database selected in the TUI.
type Client struct {
	mu            sync.Mutex
	options       goredis.Options
	clients       map[int]*goredis.Client
	databaseCount int
	host          string
	closed        bool
}

// Connect validates the connection and discovers its logical databases.
func Connect(ctx context.Context, settings Settings) (*Client, error) {
	options, host, err := connectionOptions(settings)
	if err != nil {
		return nil, err
	}
	options.DB = 0
	base := goredis.NewClient(&options)
	if err := base.Ping(ctx).Err(); err != nil {
		_ = base.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}
	configuration, err := base.ConfigGet(ctx, "databases").Result()
	if err != nil {
		_ = base.Close()
		return nil, fmt.Errorf("discover Redis logical databases: %w", err)
	}
	count, err := configuredDatabaseCount(configuration)
	if err != nil {
		_ = base.Close()
		return nil, err
	}
	return &Client{
		options:       options,
		clients:       map[int]*goredis.Client{0: base},
		databaseCount: count,
		host:          host,
	}, nil
}

func configuredDatabaseCount(configuration map[string]string) (int, error) {
	count, err := strconv.Atoi(configuration["databases"])
	if err != nil || count < 1 {
		return 0, errors.New("Redis returned an invalid logical database count")
	}
	return count, nil
}

func connectionOptions(settings Settings) (goredis.Options, string, error) {
	if dsn := strings.TrimSpace(settings.DSN); dsn != "" {
		parsed, err := url.Parse(dsn)
		if err != nil || parsed.Scheme != "redis" || parsed.Hostname() == "" {
			return goredis.Options{}, "", errors.New("valid redis:// DSN is required")
		}
		options, err := goredis.ParseURL(dsn)
		if err != nil || options.DB != 0 {
			return goredis.Options{}, "", errors.New("valid redis:// DSN for db0 is required")
		}
		return *options, parsed.Hostname(), nil
	}
	host := strings.TrimSpace(settings.Host)
	if host == "" {
		return goredis.Options{}, "", errors.New("host is required")
	}
	if settings.Port < 1 || settings.Port > 65535 {
		return goredis.Options{}, "", errors.New("port must be between 1 and 65535")
	}
	return goredis.Options{
		Addr:     net.JoinHostPort(host, strconv.Itoa(settings.Port)),
		Username: strings.TrimSpace(settings.Username),
		Password: settings.Password,
	}, host, nil
}

// Host returns the configured network host.
func (c *Client) Host() string { return c.host }

// DatabaseCount returns the configured number of logical databases.
func (c *Client) DatabaseCount() int { return c.databaseCount }

// PopulatedDatabases returns the logical databases currently holding keys.
func (c *Client) PopulatedDatabases(ctx context.Context) ([]int, error) {
	client, err := c.clientFor(0)
	if err != nil {
		return nil, err
	}
	info, err := client.Info(ctx, "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("discover populated Redis databases: %w", err)
	}
	return parsePopulatedDatabases(info, c.databaseCount)
}

func parsePopulatedDatabases(info string, databaseCount int) ([]int, error) {
	var databases []int
	for _, line := range strings.Split(info, "\n") {
		name, fields, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found || !strings.HasPrefix(name, "db") {
			continue
		}
		database, err := strconv.Atoi(strings.TrimPrefix(name, "db"))
		if err != nil || database < 0 || database >= databaseCount {
			continue
		}
		for _, field := range strings.Split(fields, ",") {
			key, value, found := strings.Cut(field, "=")
			if !found || key != "keys" {
				continue
			}
			count, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Redis key count for db%d: %w", database, err)
			}
			if count > 0 {
				databases = append(databases, database)
			}
			break
		}
	}
	slices.Sort(databases)
	return databases, nil
}

func (c *Client) clientFor(database int) (*goredis.Client, error) {
	if database < 0 || database >= c.databaseCount {
		return nil, fmt.Errorf("Redis database %d is out of range", database)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("Redis connection is closed")
	}
	if client := c.clients[database]; client != nil {
		return client, nil
	}
	options := c.options
	options.DB = database
	client := goredis.NewClient(&options)
	c.clients[database] = client
	return client, nil
}

// Close releases every logical database client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	var result error
	for _, client := range c.clients {
		result = errors.Join(result, client.Close())
	}
	return result
}

// LoadKeyPage reads one page from a logical database in Redis scan order.
func (c *Client) LoadKeyPage(ctx context.Context, database, limit int, cursor ScanCursor) (KeyPage, error) {
	client, err := c.clientFor(database)
	if err != nil {
		return KeyPage{}, err
	}
	if limit < 1 {
		return KeyPage{}, fmt.Errorf("Redis key limit must be positive: %d", limit)
	}
	cursor.visited = maps.Clone(cursor.visited)
	if cursor.visited == nil {
		cursor.visited = make(map[string]struct{})
	}
	keys := make([]Key, 0, limit+1)
	if cursor.ready != nil {
		keys = append(keys, *cursor.ready)
		cursor.ready = nil
	}
	for len(keys) <= limit {
		if err := ctx.Err(); err != nil {
			return KeyPage{}, err
		}
		if len(cursor.pending) == 0 {
			if cursor.done {
				break
			}
			var names []string
			var next uint64
			names, next, err = client.Scan(ctx, cursor.scan, "*", 0).Result()
			if err != nil {
				return KeyPage{}, fmt.Errorf("scan Redis keys: %w", err)
			}
			cursor.pending = names
			cursor.scan = next
			cursor.done = next == 0
			continue
		}
		name := cursor.pending[0]
		cursor.pending = cursor.pending[1:]
		if _, seen := cursor.visited[name]; seen {
			continue
		}
		cursor.visited[name] = struct{}{}
		key, exists, err := loadKey(ctx, client, name)
		if err != nil {
			return KeyPage{}, err
		}
		if exists {
			keys = append(keys, key)
		}
	}
	if len(keys) > limit {
		ready := keys[limit]
		cursor.ready = &ready
		return KeyPage{Keys: keys[:limit], Next: cursor, HasMore: true}, nil
	}
	return KeyPage{Keys: keys, Next: cursor}, nil
}

func loadKey(ctx context.Context, client *goredis.Client, name string) (Key, bool, error) {
	kind, err := client.Type(ctx, name).Result()
	if err != nil {
		return Key{}, false, fmt.Errorf("read Redis key type: %w", err)
	}
	if kind == "none" {
		return Key{}, false, nil
	}
	value, err := readValue(ctx, client, name, kind)
	if errors.Is(err, goredis.Nil) {
		return Key{}, false, nil
	}
	if err != nil {
		return Key{}, false, fmt.Errorf("read Redis %s value: %w", kind, err)
	}
	ttl, err := client.PTTL(ctx, name).Result()
	if err != nil {
		return Key{}, false, fmt.Errorf("read Redis key TTL: %w", err)
	}
	if ttl == -2 {
		return Key{}, false, nil
	}
	return Key{Name: safeText(name), Value: value, Type: kind, TTL: formatTTL(ttl)}, true, nil
}

func readValue(ctx context.Context, client *goredis.Client, name, kind string) (string, error) {
	var value any
	switch kind {
	case "string":
		stringValue, err := client.Get(ctx, name).Result()
		if err != nil {
			return "", err
		}
		return safeText(stringValue), nil
	case "hash":
		fields, getErr := client.HGetAll(ctx, name).Result()
		if getErr != nil {
			return "", getErr
		}
		formatted := make(map[string]string, len(fields))
		for field, fieldValue := range fields {
			formatted[safeText(field)] = safeText(fieldValue)
		}
		value = formatted
	case "list":
		members, getErr := client.LRange(ctx, name, 0, -1).Result()
		if getErr != nil {
			return "", getErr
		}
		for index := range members {
			members[index] = safeText(members[index])
		}
		value = members
	case "set":
		members, err := client.SMembers(ctx, name).Result()
		if err != nil {
			return "", err
		}
		sort.Strings(members)
		for index := range members {
			members[index] = safeText(members[index])
		}
		value = members
	case "zset":
		entries, err := client.ZRangeWithScores(ctx, name, 0, -1).Result()
		if err != nil {
			return "", err
		}
		formatted := make([]struct {
			Member string  `json:"member"`
			Score  float64 `json:"score"`
		}, len(entries))
		for index, entry := range entries {
			formatted[index].Member = safeText(fmt.Sprint(entry.Member))
			formatted[index].Score = entry.Score
		}
		value = formatted
	case "stream":
		messages, getErr := client.XRange(ctx, name, "-", "+").Result()
		if getErr != nil {
			return "", getErr
		}
		formatted := make([]struct {
			ID     string            `json:"id"`
			Values map[string]string `json:"values"`
		}, len(messages))
		for index, message := range messages {
			formatted[index].ID = safeText(message.ID)
			formatted[index].Values = make(map[string]string, len(message.Values))
			for field, fieldValue := range message.Values {
				formatted[index].Values[safeText(field)] = safeText(fmt.Sprint(fieldValue))
			}
		}
		value = formatted
	default:
		return "", fmt.Errorf("unsupported Redis key type %q", kind)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("format Redis value: %w", err)
	}
	return safeText(string(encoded)), nil
}

func formatTTL(ttl time.Duration) string {
	if ttl < 0 {
		return "—"
	}
	return ttl.String()
}

func safeText(value string) string {
	if !utf8.ValidString(value) {
		return fmt.Sprintf("0x%x", []byte(value))
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return strconv.QuoteToASCII(value)
		}
	}
	return value
}

// Execute runs one finite command on the selected logical database.
func (c *Client) Execute(ctx context.Context, database int, command string) (Result, error) {
	arguments, err := parseCommand(command)
	if err != nil {
		return Result{}, err
	}
	client, err := c.clientFor(database)
	if err != nil {
		return Result{}, err
	}
	verb := strings.ToUpper(arguments[0])
	if verb == "SELECT" {
		if len(arguments) != 2 {
			return Result{}, errors.New("SELECT requires one database index")
		}
		selected, err := strconv.Atoi(arguments[1])
		if err != nil || selected < 0 || selected >= c.databaseCount {
			return Result{}, errors.New("Redis database index is out of range")
		}
		return Result{Raw: "OK", CommandTag: verb, SelectDatabase: &selected}, nil
	}
	args := make([]any, len(arguments))
	for index, argument := range arguments {
		args[index] = argument
	}
	reply, err := client.Do(ctx, args...).Result()
	if errors.Is(err, goredis.Nil) {
		return Result{Raw: "(nil)", CommandTag: verb}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("execute Redis command: %w", err)
	}
	return Result{Raw: rawReply(reply), CommandTag: verb}, nil
}

func parseCommand(command string) ([]string, error) {
	if strings.ContainsAny(command, "\r\n") {
		return nil, errors.New("enter one Redis command at a time")
	}
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	started := false
	for _, character := range command {
		switch {
		case escaped:
			current.WriteRune(character)
			escaped = false
		case character == '\\':
			escaped = true
			started = true
		case quote != 0 && character == quote:
			quote = 0
		case quote == 0 && (character == '\'' || character == '"'):
			quote = character
			started = true
		case quote == 0 && unicode.IsSpace(character):
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(character)
			started = true
		}
	}
	if quote != 0 || escaped {
		return nil, errors.New("unfinished Redis command quote or escape")
	}
	if started {
		args = append(args, current.String())
	}
	if len(args) == 0 {
		return nil, errors.New("Redis command is empty")
	}
	return args, nil
}

type rawEntry struct {
	key   string
	value any
}

func rawReply(reply any) string {
	switch value := reply.(type) {
	case nil:
		return "(nil)"
	case string:
		return safeRawString(value)
	case []byte:
		return safeRawString(string(value))
	case []any:
		if len(value) == 0 {
			return "(empty)"
		}
		lines := make([]string, len(value))
		for index, item := range value {
			lines[index] = rawReply(item)
		}
		return strings.Join(lines, "\n")
	case map[any]any:
		entries := make([]rawEntry, 0, len(value))
		for key, item := range value {
			entries = append(entries, rawEntry{key: safeRawString(fmt.Sprint(key)), value: item})
		}
		return rawMap(entries)
	case map[string]any:
		entries := make([]rawEntry, 0, len(value))
		for key, item := range value {
			entries = append(entries, rawEntry{key: safeRawString(key), value: item})
		}
		return rawMap(entries)
	case map[string]string:
		entries := make([]rawEntry, 0, len(value))
		for key, item := range value {
			entries = append(entries, rawEntry{key: safeRawString(key), value: item})
		}
		return rawMap(entries)
	default:
		return safeRawString(fmt.Sprint(value))
	}
}

func rawMap(entries []rawEntry) string {
	if len(entries) == 0 {
		return "(empty)"
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	lines := make([]string, len(entries))
	for index, entry := range entries {
		lines[index] = entry.key + ": " + rawReply(entry.value)
	}
	return strings.Join(lines, "\n")
}

func safeRawString(value string) string {
	if !utf8.ValidString(value) {
		return fmt.Sprintf("0x%x", []byte(value))
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.Map(func(character rune) rune {
		if character == '\n' {
			return character
		}
		if unicode.IsControl(character) {
			return '�'
		}
		return character
	}, value)
}
