// backup creates a consistent online SQLite snapshot using VACUUM INTO.
package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	src, dest := os.Getenv("SKILLTREE_DB"), os.Getenv("SKILLTREE_BACKUP_OUTPUT")
	if src == "" || dest == "" {
		return fmt.Errorf("SKILLTREE_DB and SKILLTREE_BACKUP_OUTPUT are required")
	}
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	f.Close()
	success := false
	defer func() {
		if !success {
			os.Remove(dest)
		}
	}()
	db, err := sql.Open("sqlite", readOnlyURI(src))
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err = db.Exec(`VACUUM INTO ?`, dest); err != nil {
		return err
	}
	backup, err := sql.Open("sqlite", readOnlyURI(dest))
	if err != nil {
		return err
	}
	defer backup.Close()
	var result string
	if err = backup.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil || result != "ok" {
		return fmt.Errorf("backup integrity check failed")
	}
	success = true
	fmt.Println("backup integrity: ok")
	return nil
}

func readOnlyURI(path string) string {
	abs, _ := filepath.Abs(path)
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p}
	q := u.Query()
	q.Set("mode", "ro")
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	return u.String()
}
