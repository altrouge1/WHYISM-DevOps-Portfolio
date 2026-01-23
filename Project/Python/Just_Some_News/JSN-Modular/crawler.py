# crawler.py
import requests
# [보안 수정] 표준 라이브러리 대신 defusedxml 사용 (XXE 취약점 방어)
# import xml.etree.ElementTree as ET  <-- 기존 코드 (주석 처리)
import defusedxml.ElementTree as ET   # <-- 수정된 코드
import logging
from email.utils import parsedate_to_datetime
from datetime import datetime
import config

# 뉴스 수집 및 DB 저장 함수
def collect_news(conn):
    cursor = conn.cursor()
    try:
        # 1) Fetch RSS feed(뉴스 피드 가져오기)
        logging.info(f">>> Collecting news from RSS feed: {config.RSS_URL}...")
        response = requests.get(config.RSS_URL, timeout=10)

        # 2) Check response status(응답 상태 확인)
        if response.status_code != 200:
            logging.error(f">>>Failed to fetch RSS feed. Status code: {response.status_code}")
            return
        
        # 3) Decode EUC-KR encoded content if necessary(인코딩 처리)
        raw_data = response.content.decode('euc-kr', errors='ignore')
        clean_data = raw_data.replace('encoding="euc-kr"', '')
        
        # [보안] defusedxml이 여기서 악성 태그를 자동으로 무력화합니다.
        root = ET.fromstring(clean_data)

        # 4) Parse XML and extract items(XML 파싱 및 아이템 추출)
        items = root.findall('.//item')
        
        # 5) 쿼리 준비 (INSERT IGNORE 제거 -> 일반 INSERT 사용)
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
            # [추가된 로직] DB에 이미 존재하는 링크인지 먼저 확인 (ID Gap 방지)
            # ------------------------------------------------------------------
            cursor.execute(check_query, (link,))
            if cursor.fetchone():
                # 이미 존재하면 건너뜀
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
            
            # DB 저장 실행
            cursor.execute(insert_query, (title, link, pubDate, description))
            new_cnt += 1
            logging.info(f">>> New Article Collected: {title} (Date: {pubDate})")

        # 8) Final log summary(최종 로그 요약)
        logging.info(f">>> News Collection Completed: {new_cnt} new articles added and {len(items)} Scan Finished.")
    
    # 9) 뉴스 RSS 수집 중 error 처리
    except Exception as e:
        logging.error(f">>> Error RSSfeed Crawling Process: {e}")