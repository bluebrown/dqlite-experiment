package tools

import (
	_ "github.com/golang-migrate/migrate/v4/cmd/migrate"
	_ "github.com/volatiletech/sqlboiler/v4"
	_ "github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-sqlite3"
)

// need to install run manually: go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-sqlite3

//go:generate rm -f sql/dev.db && touch sql/dev.db
//go:generate go run -tags sqlite3 github.com/golang-migrate/migrate/v4/cmd/migrate -source file://sql/migrations -database sqlite3://sql/dev.db up
//go:generate go run github.com/volatiletech/sqlboiler/v4 --config sql/boil.toml sqlite3
