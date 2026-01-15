# 프로그램 이름: Just Some News
# 이름의 이유: 뉴스를 보면서 최신 소식은 무조건 읽어야 한다는 의미로 지은 이름
# 작성자: whyism

# DB 테이블 설계
# - MariaDB에 수집된 뉴스를 저장
# - DB는 기사별 고유 link (URL)를 기준으로 무결성 유지됨.
# - 시스템 엔지니어링 모드 포함 (DB 및 테이블 생성)

import os
import sys
import mariadb
import requests
# [보안 Fix 1] 표준 라이브러리 대신 defusedxml 사용 (XXE 취약점 방어)
# pip install defusedxml 명령어로 설치 필요
import defusedxml.ElementTree as ET 
import logging
from logging.handlers import TimedRotatingFileHandler
from email.utils import parsedate_to_datetime
from datetime import datetime

# ==========================================
# 1. 설정 정보 (Configuration)
# ==========================================

# 접속 정보
# [보안 Note] 실제 운영 환경에서는 환경변수(os.getenv) 사용을 권장합니다.
DB_HOST = '192.168.1.46'
DB_PORT = 3306
DB_USER = 'rl'
DB_PASSWORD = 'rockylinux'

TARGET_DB_NAME = 'read_news'

# [보안 Fix 2] HTTP -> HTTPS 변경 (전송 구간 암호화)
RSS_URL = "https://www.boannews.com/media/news_rss.xml"

# ==========================================
# 2. 로깅 설정 (Logging Configuration)
# ==========================================

# 기본 경로: 현재 실행 위치의 logs 폴더 / jsn.log
DEFAULT_LOG_DIR = "logs"
DEFAULT_LOG_FILE = "jsn.log"

# 인자가 있으면 그걸 쓰고, 없으면 기본값 사용
if len(sys.argv) > 1:
    log_file_path = sys.argv[1]
else:
    # 실행되는 파일이 있는 위치를 기준으로 경로 설정 (안전빵)
    base_dir = os.path.dirname(os.path.abspath(__file__))
    log_dir = os.path.join(base_dir, DEFAULT_LOG_DIR)
    log_file_path = os.path.join(log_dir, DEFAULT_LOG_FILE)

# [핵심 Fix] 로그 파일이 위치할 '폴더'가 없으면 무조건 생성
log_dir_path = os.path.dirname(log_file_path)
if not os.path.exists(log_dir_path):
    os.makedirs(log_dir_path, exist_ok=True)
    print(f">>> Log directory created: {log_dir_path}")

# 로거 생성
logger = logging.getLogger()
logger.setLevel(logging.INFO)

# 핸들러 설정 (파일명은 고정, 자정마다 날짜 붙여서 백업)
file_handler = TimedRotatingFileHandler(
    filename=log_file_path, 
    when='midnight', 
    interval=1, 
    backupCount=14, 
    encoding='utf-8'
)
file_handler.suffix = "%Y%m%d" # 백업 파일명 뒤에 붙을 날짜 포맷

# 포맷터 설정
formatter = logging.Formatter('[%(levelname)s] %(asctime)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S')
file_handler.setFormatter(formatter)
logger.addHandler(file_handler)

# 콘솔 출력용 핸들러 추가
console_handler = logging.StreamHandler()
console_handler.setFormatter(formatter)
logger.addHandler(console_handler)

# ==========================================
# 3. 함수 정의 (Function Definitions)
# ==========================================
# 시스템 엔지니어링 모드: DB 및 테이블 생성
def initialize_infrastructure():
    '''
    [시스템 엔지니어링 모드]
    Connect DB -> Checking Permission -> Create DB/Tables if not exist
    '''
    conn = None
    try:
        # 1. First, connect to MariaDB server without specifying a database(DB 초기 접속)
        logging.info(">>> System Initializing: Connecting to MariaDB server...")
        conn = mariadb.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD
        )
        conn.autocommit = True
        cursor = conn.cursor()

        # 2. Check if we can create databases and tables(권한 확인)
        logging.info(">>> Checking JOBs: Checking permissions to create database and tables...")
        cursor.execute("SHOW GRANTS FOR CURRENT_USER()")
        grants = [row[0] for row in cursor.fetchall()]

        # Recoding permission logs(권한 로그 기록)
        has_create_priv = False
        for grant in grants:
            logging.info(f"[Grant Check] {grant}")
            if "ALL PRIVILEGES" in grant or "CREATE" in grant:
                has_create_priv = True
        if not has_create_priv:
            logging.warning(">>> Permission Denied: User lacks CREATE privileges.")
        else:
            logging.info(">>> Permission Granted: User has CREATE privileges.")
        
        # 3. Create database if not exists(DB 생성)
        # [참고] DB 이름 식별자는 바인딩 불가하므로 f-string 사용 (내부 설정값이므로 안전)
        logging.info(f">>> Creating Database if not exists: {TARGET_DB_NAME}...")
        cursor.execute(f"CREATE DATABASE IF NOT EXISTS {TARGET_DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;")

        # 4. Use the target database(DB 선택)
        conn.select_db(TARGET_DB_NAME)

        # 5. Create necessary tables if not exists(스키마 정의 및 테이블 생성)
        logging.info(">>> Checking Table schema and Creating if not exists...")
        create_table_queries = """
        CREATE TABLE IF NOT EXISTS security_articles (
            id INT AUTO_INCREMENT PRIMARY KEY,
            title VARCHAR(512) NOT NULL,
            link VARCHAR(1024) NOT NULL UNIQUE COMMENT 'Unique link to the article',
            pubDate DATETIME NOT NULL,
            description TEXT,
            collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)
            ENGINE=InnoDB COMMENT='Security News Articles';
        """
        cursor.execute(create_table_queries)
        logging.info(">>> Infrastructure Initialization Completed Successfully.")
        return conn
    # 6. Exception handling(예외 처리)
    except mariadb.Error as e:
        logging.error(f">>> Error during infrastructure initialization: {e}")
        sys.exit(1)

# 뉴스 수집 및 DB 저장 함수
def collect_news(conn):
    cursor = conn.cursor()
    try:
        # 1) Fetch RSS feed(뉴스 피드 가져오기)
        logging.info(f">>> Collecting news from RSS feed: {RSS_URL}...")
        response = requests.get(RSS_URL, timeout=10)

        # 2) Check response status(응답 상태 확인)
        if response.status_code != 200:
            logging.error(f">>>Failed to fetch RSS feed. Status code: {response.status_code}")
            return
        
        # 3) Decode EUC-KR encoded content if necessary(인코딩 처리)
        raw_data = response.content.decode('euc-kr', errors='ignore')
        clean_data = raw_data.replace('encoding="euc-kr"', '')
        
        # [보안 Fix 1 적용] defusedxml이 악성 태그를 무력화함
        root = ET.fromstring(clean_data)

        # 4) Parse XML and extract items(XML 파싱 및 아이템 추출)
        items = root.findall('.//item')
        
        # 5) 쿼리 준비 (INSERT IGNORE 제거 -> 일반 INSERT 사용)
        # [보안] 데이터 값에는 반드시 ? 홀더를 사용하여 SQL Injection 방지
        check_query = "SELECT id FROM security_articles WHERE link = ?"
        insert_query = "INSERT INTO security_articles (title, link, pubDate, description) VALUES (?, ?, ?, ?)"

        # 6) Track new articles count(신규 기사 카운트)
        new_cnt = 0
        logging.info(f">>> Processing {len(items)} items from RSS feed...")

        # Dublin Core 네임스페이스 정의
        ns = {'dc': 'http://purl.org/dc/elements/1.1/'}

        # 7) Loop through items and insert into DB(아이템 루프 및 DB 삽입)
        for item in items:
            title = item.find('title').text if item.find('title') is not None else "No Title"
            link = item.find('link').text if item.find('link') is not None else ""
            
            # ------------------------------------------------------------------
            # [기능 Fix] DB에 이미 존재하는 링크인지 먼저 확인 (ID Gap 방지)
            # ------------------------------------------------------------------
            cursor.execute(check_query, (link,))
            if cursor.fetchone():
                # 이미 존재하면 건너뜀 (번호표를 뽑지 않음)
                continue

            # ------------------------------------------------------------------
            # 존재하지 않는다면(신규), 아래 날짜 파싱 및 저장 로직 실행
            # ------------------------------------------------------------------

            # [Date Logic Fix] dc:date 태그 타격 및 RFC 2822 파싱
            dc_date_elem = item.find('dc:date', ns)
            pubDate = None
            
            if dc_date_elem is not None and dc_date_elem.text:
                raw_date = dc_date_elem.text.strip()
                try:
                    dt = parsedate_to_datetime(raw_date)
                    pubDate = dt.strftime('%Y-%m-%d %H:%M:%S')
                except Exception:
                    pass
            
            # [Fallback]
            if pubDate is None:
                pubDate = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            
            description = item.find('description').text if item.find('description') is not None else ""
            
            # [저장 실행] 여기서 비로소 번호표(ID)가 1 증가합니다.
            cursor.execute(insert_query, (title, link, pubDate, description))
            new_cnt += 1
            logging.info(f">>> New Article Collected: {title} (Date: {pubDate})")

        # 8) Final log summary(최종 로그 요약)
        logging.info(f">>> News Collection Completed: {new_cnt} new articles added and {len(items)} Scan Finished.")
    
    # 9) 뉴스 RSS 수집 중 error 처리
    except Exception as e:
        logging.error(f">>> Error RSSfeed Crawling Process: {e}")

# ==========================================
# 4. 메인 프로그램 실행 (Main Program Execution)
# ==========================================

if __name__ == "__main__":
    # 1. Initialize Infrastructure(인프라 초기화)
    connection = initialize_infrastructure()

    # 2. Close DB connection(DB 연결 종료)
    if connection:
        collect_news(connection)
        connection.close()
        logging.info(">>> Database connection closed. Program finished.")