package db

import (
	"database/sql"
	"fmt"
	"jsn-modular/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func InitDB() (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// DB 생성 및 테이블 설정 (모놀리딕 코드의 로직 이식)
	_, _ = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4", config.DBName))
	_, _ = db.Exec(fmt.Sprintf("USE %s", config.DBName))

	tableQuery := `
    CREATE TABLE IF NOT EXISTS security_articles (
        id INT AUTO_INCREMENT PRIMARY KEY,
        title VARCHAR(512) NOT NULL,
        link VARCHAR(1024) NOT NULL UNIQUE,
        pubDate DATETIME NOT NULL,
        description TEXT,
        collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    ) ENGINE=InnoDB;`

	_, err = db.Exec(tableQuery)
	return db, err
}
