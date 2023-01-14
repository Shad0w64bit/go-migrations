package migrations

import (
	"database/sql"
	"errors"
	"log"
	"sort"
	"time"
)

type Config struct {
	// Number of migrations applied at one time
	Step int

	// Path with migrations
	Path string

	// Database connection
	Db *sql.DB

	// Output to console
	// 0 - No output
	// 1 - Verbose
	// 2 - Over Verbose
	Verbose int
	Timeout time.Duration
}

var config = Config{
	Step:    -1,
	Path:    "./migrations/",
	Verbose: 1,
	Timeout: 5 * time.Second,
}

type migrationRecord struct {
	Time time.Time
	Name string
}

func GetConfig() Config {
	return config
}

func SetConfig(cfg Config) {
	config = cfg
}

// Migrate all (Step==-1) or step()
func Up() error {
	log.Print("=== Migration Up ===")
	if config.Db == nil || config.Db.Ping() != nil {
		return errors.New("Migration: Database unreachable")
	}
	// 1. Получаем список миграций
	pathMigrationList, err := getPathMigrations(config.Path)
	if err != nil {
		return err
	}

	if config.Verbose > 1 {
		log.Print("=== All migration ===")

		if len(pathMigrationList) == 0 {
			log.Print("(nothing)")
		}

		for _, item := range pathMigrationList {
			log.Println(item.Name)
		}

		log.Print("=== End ===")
	}


	// 2 Создать таблицу для миграций если нужно
	if err := createMigrationTableIfNeed(config.Db); err != nil {
		return err
	}

	// 3. Получаем уже примененые миграции из БД
	dbMigrationList, err := getDatabaseMigrations(config.Db)
	if err != nil {
		return err
	}

	if config.Verbose > 1 {
		log.Print("=== Found in database ===")

		if len(dbMigrationList) == 0 {
			log.Print("(nothing)")
		}

		for _, item := range dbMigrationList {
			log.Println(item.Name)
		}

		log.Print("=== End ===")
	}

	// 4. Сортируем по времени от старого к новому
	sort.Slice(dbMigrationList, func(i, j int) bool {
		return dbMigrationList[i].Time.Before(dbMigrationList[j].Time)
	})

	// 5. Определяем разницу
	selMigrations, err := diffMigrations(pathMigrationList, dbMigrationList)
	if err != nil {
		return err
	}

	// 6. Отрезаем N-миграций если указан Step
	if config.Step > 0 && config.Step < len(selMigrations) {
		selMigrations = selMigrations[:config.Step]
	}

	if config.Verbose > 1 {
		log.Print("=== Run migrations ===")

		if len(selMigrations) == 0 {
			log.Print("(nothing)")
		}

		for _, item := range selMigrations {
			log.Println(item.Name)
		}

		log.Print("=== End ===")
	}

	// 7. Пытаемся накатить миграции которых ещё нет
	for _, migrate := range selMigrations {
		log.Printf("Run migration: %d_%s", migrate.Time.Unix(), migrate.Name)
		if err := upMigrate(&config, migrate); err != nil {
			return err
		}
	}

	// 8. Если миграций не осталось и все успешно выходим без ошибок
	if config.Verbose > 0 {
		log.Print("All migrations successfully applied")
	}
	return nil
}

func Down() error {
	log.Print("=== Migration Down ===")
	if config.Db == nil || config.Db.Ping() != nil {
		return errors.New("Migration: Database unreachable")
	}

	// 1. Создать таблицу для миграций если нужно
	if err := createMigrationTableIfNeed(config.Db); err != nil {
		return err
	}

	// 2. Получаем уже примененые миграции из БД
	selMigrations, err := getDatabaseMigrations(config.Db)
	if err != nil {
		return err
	}

	if config.Verbose > 1 {
		log.Print("=== Found in database ===")

		if len(selMigrations) == 0 {
			log.Print("(nothing)")
		}

		for _, item := range selMigrations {
			log.Println(item.Name)
		}

		log.Print("=== End ===")
	}

	// 3. Сортируем по времени от нового к старому
	sort.Slice(selMigrations, func(i, j int) bool {
		return selMigrations[i].Time.After(selMigrations[j].Time)
	})

	// 4. Отрезаем N-миграций если указан Step
	if config.Step > 0 && config.Step < len(selMigrations) {
		selMigrations = selMigrations[:config.Step]
	}

	if config.Verbose > 1 {
		log.Print("=== Revert migrations ===")

		if len(selMigrations) == 0 {
			log.Print("(nothing)")
		}

		for _, item := range selMigrations {
			log.Println(item.Name)
		}

		log.Print("=== End ===")
	}

	// 5. Пытаемся накатить миграции которых ещё нет
	for _, migrate := range selMigrations {
		log.Printf("Revert migration: %d_%s", migrate.Time.Unix(), migrate.Name)
		if err := downMigrate(&config, migrate); err != nil {
			return err
		}
	}

	// 6. Если миграций не осталось и все успешно выходим без ошибок
	if config.Verbose > 0 {
		log.Print("All migrations successfully reverted")
	}
	return nil
}

