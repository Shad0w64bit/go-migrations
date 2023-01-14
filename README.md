Каждый файл миграции должен соответсвует шаблону  
```
<id>_<name>.up.sql
<id>_<name>.down.sql

Пример:
123_create_user_table.up.sql  
123_create_user_table.down.sql  
---  
1673602735_init.up.sql  
```

id - любое положительное число, может быть как порядковым так и unixtime, но уникальным (
Является primary-key для таблицы)  
name - человеко-понятное название из слов разделенных знаком '_'

## Up
При запуске миграции определяется id последней миграции для БД (читается из schema_migrations) и накатываются все файлы <id>_<name>.up.sql следующие за ним по порядку.

Миграции id которых ниже чем у последней примененной будут пропущены.
Для их корретного запуска либо выполните Down() нужное количетсво раз, либо вручную откатите изменния в БД до необходимой миграции, после чего можно продолжить накатывать миграции с помощью Up().

## Down
Для отката изменений воспользуйтесь функцией Down()
Функция Down выгружает из таблицы schema_migrations список миграций (примененых).
Сортирует их в обратном порядке, после чего ищет файл с названием <id>_<name>.down.sql и пытается его вызвать.

## Config
```
migrations.Config struct {	
	Step int
	Path string
	Db *sql.DB
	Verbose int
}
```

Step - количество миграций которые будут применены при вызове Up() или Down() (step <= 0 - все миграции)  
Path - путь к директории содержаший файлы миграций  
Db - указатель на соединение в БД в которую необходимо раскатить миграци  
Verbose - уровень вывода логов в консоль  
	0 - не выводить логи  
	1 - только важные логи  
	2 - все логи (подробно, debug-режим)  

## Использование
```
import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	
	// migrations "github.com/Shad0w64bit/go-migrations"
	_ "github.com/Shad0w64bit/go-migrations"		

)

func main() {
	db, _ := Open()
	
	cfg := migrations.GetConfig()
	cfg.Db = db
	cfg.Path = "./migrations"
	cfg.Verbose = 1 	
	migrations.SetConfig( cfg )
	
	# Migrate Up
	if err := migrations.Up(); err != nil {	
		log.Fatal(err)
	}
	
	# Migrate Down
	if err := migrations.Up(); err != nil {	
		log.Fatal(err)
	}
}
