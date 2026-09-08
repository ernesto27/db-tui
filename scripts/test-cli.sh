#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
dsn="postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"
mysql_dsn="mysql://db_tui:db_tui@127.0.0.1:3307/world"
oracle_dsn="oracle://db_tui:db_tui@127.0.0.1:1522/FREEPDB1"
sqlserver_dsn="sqlserver://sa:DbTuiSql2026%21@127.0.0.1:1434?database=db_tui&encrypt=true&trustservercertificate=true"
sqlite_path="$repo_root/docker/sqlite/employee.db"

temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT
binary="$temp_dir/db-tui"

fail() {
	echo "FAIL: $1" >&2
	exit 1
}

expect_cli() {
	local name="$1"
	local expected_exit="$2"
	local expected_stdout="$3"
	local expected_stderr="$4"
	shift 4

	local stdout_file="$temp_dir/$name.stdout"
	local stderr_file="$temp_dir/$name.stderr"
	local actual_exit

	if "$binary" "$@" >"$stdout_file" 2>"$stderr_file"; then
		actual_exit=0
	else
		actual_exit=$?
	fi

	local actual_stdout
	local actual_stderr
	actual_stdout="$(<"$stdout_file")"
	actual_stderr="$(<"$stderr_file")"

	[[ "$actual_exit" == "$expected_exit" ]] ||
		fail "$name exit code is $actual_exit; want $expected_exit"
	[[ "$actual_stdout" == "$expected_stdout" ]] ||
		fail "$name stdout is $actual_stdout; want $expected_stdout"

	if [[ "$expected_stderr" == "*" ]]; then
		[[ -n "$actual_stderr" ]] || fail "$name stderr is empty"
	elif [[ "$actual_stderr" != "$expected_stderr" ]]; then
		fail "$name stderr is $actual_stderr; want $expected_stderr"
	fi

	echo "PASS: $name"
}

cd "$repo_root"
docker compose up -d --wait postgres mysql oracle sqlserver
go build -o "$binary" ./cmd/db-tui

expect_cli \
	"postgres_two_rows" \
	0 \
	$'[\n  {\n    "ArtistId": 1,\n    "Name": "AC/DC"\n  },\n  {\n    "ArtistId": 2,\n    "Name": "Accept"\n  }\n]' \
	"" \
	-q 'SELECT "ArtistId", "Name" FROM public."Artist" ORDER BY "ArtistId" LIMIT 2' \
	-c "$dsn"

expect_cli \
	"postgres_empty_result" \
	0 \
	"[]" \
	"" \
	-q 'SELECT "ArtistId" FROM public."Artist" WHERE "ArtistId" = -1' \
	-c "$dsn"

expect_cli \
	"postgres_json_normalization" \
	0 \
	$'[\n  {\n    "Identifier": "3234b411-89ab-4cde-8f01-23456789abcd",\n    "Measurement": "NaN",\n    "RecordedAt": "infinity"\n  }\n]' \
	"" \
	-q 'SELECT "Identifier", "Measurement", "RecordedAt" FROM public."CLIJSONExample"' \
	-c "$dsn"

expect_cli \
	"missing_dsn" \
	2 \
	"" \
	"db-tui: -q and -c must be provided together" \
	-q 'SELECT 1'

expect_cli \
	"missing_query" \
	2 \
	"" \
	"db-tui: -q and -c must be provided together" \
	-c "$dsn"

expect_cli \
	"unsupported_dsn" \
	2 \
	"" \
	"db-tui: unsupported DSN" \
	-q 'SELECT 1' \
	-c 'unsupported://example'

expect_cli \
	"postgres_non_select_query" \
	1 \
	"" \
	"db-tui: query: only SELECT queries can be exported" \
	-q 'UPDATE public."Artist" SET "Name" = "unchanged" WHERE false' \
	-c "$dsn"

expect_cli \
	"postgres_unreachable_database" \
	1 \
	"" \
	"*" \
	-q 'SELECT 1' \
	-c 'postgres://db_tui@127.0.0.1:1/chinook?sslmode=disable&connect_timeout=1'

expect_cli \
	"sqlite_two_rows" \
	0 \
	$'[\n  {\n    "emp_no": 10001,\n    "first_name": "Georgi"\n  },\n  {\n    "emp_no": 10002,\n    "first_name": "Bezalel"\n  }\n]' \
	"" \
	-q 'SELECT emp_no, first_name FROM employee ORDER BY emp_no LIMIT 2' \
	-c "$sqlite_path"

expect_cli \
	"sqlite_empty_result" \
	0 \
	"[]" \
	"" \
	-q 'SELECT emp_no FROM employee WHERE emp_no = -1' \
	-c "$sqlite_path"

expect_cli \
	"sqlite_non_select_query" \
	1 \
	"" \
	"db-tui: query: only SELECT queries can be exported" \
	-q 'UPDATE employee SET first_name = first_name WHERE false' \
	-c "$sqlite_path"

expect_cli \
	"mysql_two_rows" \
	0 \
	$'[\n  {\n    "ID": 1,\n    "Name": "Kabul"\n  },\n  {\n    "ID": 2,\n    "Name": "Qandahar"\n  }\n]' \
	"" \
	-q 'SELECT ID, Name FROM city ORDER BY ID LIMIT 2' \
	-c "$mysql_dsn"

expect_cli \
	"mysql_empty_result" \
	0 \
	"[]" \
	"" \
	-q 'SELECT ID FROM city WHERE ID = -1' \
	-c "$mysql_dsn"

expect_cli \
	"mysql_non_select_query" \
	1 \
	"" \
	"db-tui: query: only SELECT queries can be exported" \
	-q 'UPDATE city SET Name = Name WHERE false' \
	-c "$mysql_dsn"

expect_cli \
	"mysql_unreachable_database" \
	1 \
	"" \
	"*" \
	-q 'SELECT 1' \
	-c 'mysql://db_tui:db_tui@127.0.0.1:1/world?timeout=1s'

expect_cli \
	"oracle_json_result" \
	0 \
	$'[\n  {\n    "GREETING": "hello"\n  }\n]' \
	"" \
	-q "SELECT 'hello' AS greeting FROM dual" \
	-c "$oracle_dsn"

expect_cli \
	"oracle_non_select_query" \
	1 \
	"" \
	"db-tui: query: only SELECT queries can be exported" \
	-q 'DROP TABLE countries' \
	-c "$oracle_dsn"

expect_cli \
	"sqlserver_json_result" \
	0 \
	$'[\n  {\n    "greeting": "hello"\n  }\n]' \
	"" \
	-q "SELECT CAST('hello' AS nvarchar(5)) AS greeting" \
	-c "$sqlserver_dsn"

expect_cli \
	"sqlserver_non_select_query" \
	1 \
	"" \
	"db-tui: query: only SELECT queries can be exported" \
	-q 'DROP TABLE dbo.cities' \
	-c "$sqlserver_dsn"

echo "All CLI scenarios passed."
