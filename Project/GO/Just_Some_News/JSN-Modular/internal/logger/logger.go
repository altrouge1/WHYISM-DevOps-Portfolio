package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// Setup은 로그 디렉토리를 확인하고 파일/콘솔 멀티 로거를 설정합니다.
func Setup() *os.File {
	logDir := "logs"
	logFile := filepath.Join(logDir, "jsn.log")

	// 폴더가 없으면 생성
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.MkdirAll(logDir, 0755)
	}

	// 로그 파일 오픈
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("로그 파일 생성 실패: %v", err)
	}

	// 콘솔(Stdout)과 파일(f)에 동시 출력 설정
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)

	// 로그 포맷 설정 (날짜, 시간, 파일 라인 넘버)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return f
}
