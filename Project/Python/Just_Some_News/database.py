# database.py
import sys
import mariadb
import logging
import config

# ==========================================
# 3. DB 인프라 함수 (Infrastructure)
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
            host=config.DB_HOST,
            port=config.DB_PORT,
            user=config.DB_USER,
            password=config.DB_PASSWORD
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
        logging.info(f">>> Creating Database if not exists: {config.TARGET_DB_NAME}...")
        cursor.execute(f"CREATE DATABASE IF NOT EXISTS {config.TARGET_DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;")

        # 4. Use the target database(DB 선택)
        conn.select_db(config.TARGET_DB_NAME)

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