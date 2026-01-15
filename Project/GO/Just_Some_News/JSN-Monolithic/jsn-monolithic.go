// jsn-monolithic.go
// Just Some News - Monolithic Architecture
// 보안 뉴스를 수집하여 MariaDB에 저장하는 프로그램
// 작성자: whyism
// 작성일: 2025-01-13

package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MariaDB 드라이버
	"golang.org/x/text/encoding/korean"
)

// ==========================================
// 1. 설정 및 구조체 정의
// ==========================================

const (
	DBUser     = "rl"
	DBPassword = "rockylinux"
	DBHost     = "192.168.1.46"
	DBPort     = 3306
	DBName     = "read_news"
	RSSURL     = "https://www.boannews.com/media/news_rss.xml"
)

// RSS XML 구조 정의
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	// dc:date 네임스페이스 처리를 위해 구조 설계
	Date string `xml:"http://purl.org/dc/elements/1.1/ date"`
}

// ==========================================
// 2. 로깅 설정
// ==========================================

func setupLogger() *os.File {
	logDir := "logs"
	logFile := filepath.Join(logDir, "jsn.log")

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("로그 파일 생성 실패: %v", err)
	}

	// 표준 로그 출력을 파일과 콘솔에 동시에 설정
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return f
}

// ==========================================
// 3. 인프라 및 뉴스 수집 로직
// ==========================================

func initializeDB() (*sql.DB, error) {
	// 1. 서버 접속 (DB 미지정)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true", DBUser, DBPassword, DBHost, DBPort)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 2. DB 및 테이블 생성
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4", DBName))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(fmt.Sprintf("USE %s", DBName))
	if err != nil {
		return nil, err
	}

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

func collectNews(db *sql.DB) {
	log.Println(">>> 뉴스 수집 시작...")

	// 1. HTTP 요청
	resp, err := http.Get(RSSURL)
	if err != nil {
		log.Printf("RSS 요청 실패: %v", err)
		return
	}
	defer resp.Body.Close()

	// 2. XML 디코더 생성 (변환되지 않은 원본 resp.Body 사용)
	decoder := xml.NewDecoder(resp.Body)

	// [핵심 Fix] XML 헤더의 encoding 선언을 보고 적절한 디코더를 연결해주는 함수
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if strings.ToLower(charset) == "euc-kr" {
			// EUC-KR 리더를 반환하여 디코더가 내부적으로 UTF-8로 읽게 함
			return korean.EUCKR.NewDecoder().Reader(input), nil
		}
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}

	// 3. XML 파싱
	var rss RSS
	// 보안을 위해 필요 시 Strict 모드 조절 가능
	if err := decoder.Decode(&rss); err != nil {
		log.Printf("XML 파싱 실패: %v", err)
		return
	}

	newCnt := 0
	for _, item := range rss.Channel.Items {
		// 중복 확인
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM security_articles WHERE link = ?)", item.Link).Scan(&exists)
		if err != nil {
			log.Printf("DB 조회 에러: %v", err)
			continue
		}

		if !exists {
			// 날짜 파싱 (dc:date 형식 처리)
			// 예: 2025-05-20T15:00:00Z 또는 유사 형식
			t, err := time.Parse(time.RFC3339, item.Date)
			if err != nil {
				t = time.Now() // 실패 시 현재 시간
			}

			_, err = db.Exec(
				"INSERT INTO security_articles (title, link, pubDate, description) VALUES (?, ?, ?, ?)",
				item.Title, item.Link, t, item.Description,
			)
			if err != nil {
				log.Printf("저장 에러: %v", err)
				continue
			}
			newCnt++
			log.Printf(">>> 신규 수집: %s", item.Title)
		}
	}

	log.Printf(">>> 수집 완료: 신규 %d건 / 전체 %d건 스캔", newCnt, len(rss.Channel.Items))
}

// ==========================================
// 4. 메인 실행
// ==========================================

func main() {
	logFile := setupLogger()
	defer logFile.Close()

	db, err := initializeDB()
	if err != nil {
		log.Fatalf("인프라 초기화 실패: %v", err)
	}
	defer db.Close()

	collectNews(db)
}
