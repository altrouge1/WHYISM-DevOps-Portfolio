package main

import (
	"jsn-modular/internal/db"
	"jsn-modular/internal/logger"
	"jsn-modular/internal/rss"
	"log"
)

func main() {
	// 1. 로거 설정
	logFile := logger.Setup()
	defer logFile.Close()

	// 2. 데이터베이스 초기화
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("인프라 초기화 실패: %v", err)
	}
	defer database.Close()

	// 3. 뉴스 수집 실행
	rss.Collect(database)

	log.Println(">>> 프로그램 정상 종료.")
}
