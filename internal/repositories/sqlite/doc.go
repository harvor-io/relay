// Package sqlite provides SQLite-backed implementations of the repository
// interfaces defined in the repositories package. It depends only on
// database/sql; the caller opens the *sql.DB (see internal/database) and keeps
// the schema current (see the migrations package and `relay migrate`).
//
// Storage-encoding helpers shared with the connection layer (timestamp layout,
// NULL handling, bool/int mapping, rows-affected checks) live in
// internal/database/sqlite and are imported here as sqlitedb.
package sqlite
