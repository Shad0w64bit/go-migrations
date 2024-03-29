package migrations

import (
	"context"
	"database/sql"
//	"errors"
"fmt"
	"io/ioutil"
	"log"
//	"os"
"regexp"
	"sort"
	"strconv"
	"time"
)


func upMigrate(cfg *Config, m migrationRecord) error {
	ctx, cancel:= context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	tx, err := cfg.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (id, name) VALUES ($1, $2)",
		m.Time.Unix(), m.Name)
	if err != nil {
		return err
	}

	/*
	// Genertate timeout request
	n := rand.Intn(4)+2
	_, err = tx.ExecContext(ctx, "SELECT pg_sleep($1)", n)
	if err != nil {
		return err
	}
	*/

	file, err := ioutil.ReadFile(cfg.Path + "/" + fmt.Sprintf("%d_%s.up.sql", m.Time.Unix(), m.Name) )
	//if err != nil && !errors.Is(err, os.ErrNotExist) {
	if err != nil {
		return err
	}

	if err == nil {
		_, err = tx.ExecContext( ctx, string(file) )
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}


func downMigrate(cfg *Config, m migrationRecord) error {
	ctx, cancel:= context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	tx, err := cfg.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE id=$1", m.Time.Unix())
	if err != nil {
		return err
	}

	/*
	// Genertate timeout request
	n := rand.Intn(4)+2
	_, err = tx.ExecContext(ctx, "SELECT pg_sleep($1)", n)
	if err != nil {
		return err
	}
	*/

	file, err := ioutil.ReadFile(cfg.Path + "/" + fmt.Sprintf("%d_%s.down.sql", m.Time.Unix(), m.Name) )
	//if err != nil && !errors.Is(err, os.ErrNotExist) {
	if err != nil {	
		return err
	}

	if err == nil {
		_, err = tx.ExecContext( ctx, string(file) )
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func getPathMigrations(path string) (m []migrationRecord, err error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile("(\\d+)_(\\w+).up.sql")
	if err != nil {
		return nil, err
	}
	migrationFiles := []migrationRecord{}
	for _, file := range files {
		name := file.Name()
		match := re.FindStringSubmatch(name)
		if len(match) != 0 {

			ts, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				return nil, err
			}

			migrationFiles = append(migrationFiles, migrationRecord{
				time.Unix(ts, 0),
				string(match[2]),
			})
		}
	}

	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Time.Before(migrationFiles[j].Time)
	})

	return migrationFiles, nil
}

func getDatabaseMigrations(db *sql.DB) (m []migrationRecord, err error) {
	migrationList := []migrationRecord{}

	rows, err := db.Query("SELECT id,name FROM schema_migrations")
	if err != nil {
		log.Printf("Migration select error: %s", err)
		return []migrationRecord{}, nil
	}
	defer rows.Close()

	for rows.Next() {
		var mr migrationRecord
		var ts int64
		if err := rows.Scan(&ts, &mr.Name); err != nil {
			return nil, err
		}
		mr.Time = time.Unix(ts, 0)
		migrationList = append(migrationList, mr)
	}

	return migrationList, nil
}

func createMigrationTableIfNeed(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (id bigint PRIMARY KEY, name VARCHAR(180) NOT NULL)")
	if err != nil {
		return err
	}

	return nil
}

func diffMigrations(all []migrationRecord, cur []migrationRecord) (m []migrationRecord, err error) {
	diff := []migrationRecord{}
	if len(cur) == 0 {
		// Nothing not applied
		return all, nil
	}

	// Если миграции идут после последней примененной добавляем их в список
	last := cur[len(cur)-1]
	for _, item := range all {
		if last.Time.Before(item.Time) {
			diff = append(diff, item)
		}
	}

	return diff, nil
}

