#!/usr/bin/env bash
set -euo pipefail

sqlplus -L -s /nolog <<SQL
WHENEVER OSERROR EXIT FAILURE
WHENEVER SQLERROR EXIT SQL.SQLCODE
CONNECT ${APP_USER}/${APP_USER_PASSWORD}@//localhost/FREEPDB1
@/container-entrypoint-initdb.d/002_countries-cities-currencies.dump
EXIT
SQL
