// Package rss handles RSS feed collection and parsing.
package rss

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"jsn-modular/internal/config"

	"golang.org/x/text/encoding/korean"
)

// RSS 는 보안 뉴스 수집을 위한 XML 매핑 구조체입니다.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

// Item 은 개별 뉴스 기사 정보를 담는 구조체입니다.
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Date        string `xml:"http://purl.org/dc/elements/1.1/ date"`
}

// Collect fetches and stores RSS feed items into the database.
func Collect(db *sql.DB) {
	log.Println(">>> 뉴스 수집 시작...")

	resp, err := http.Get(config.RSSURL)
	if err != nil {
		log.Printf("RSS 요청 실패: %v", err)
		return
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	// EUC-KR 처리 핸들러 등록
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if strings.ToLower(charset) == "euc-kr" {
			return korean.EUCKR.NewDecoder().Reader(input), nil
		}
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}

	var feed RSS
	if err := decoder.Decode(&feed); err != nil {
		log.Printf("XML 파싱 실패: %v", err)
		return
	}

	newCnt := 0
	for _, item := range feed.Channel.Items {
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM security_articles WHERE link = ?)"
		_ = db.QueryRow(query, item.Link).Scan(&exists)

		if !exists {
			t, err := time.Parse(time.RFC3339, item.Date)
			if err != nil {
				t = time.Now()
			}

			_, err = db.Exec(
				"INSERT INTO security_articles (title, link, pubDate, description) VALUES (?, ?, ?, ?)",
				item.Title, item.Link, t, item.Description,
			)
			if err == nil {
				newCnt++
				log.Printf(">>> 신규 수집: %s", item.Title)
			}
		}
	}
	log.Printf(">>> 수집 완료: 신규 %d건 / 전체 %d건 스캔", newCnt, len(feed.Channel.Items))
}
