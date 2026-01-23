# 프로그램 이름: Just Some News

import logging
import logger
import database
import crawler

# ==========================================
# 4. 메인 프로그램 실행 (Main Program Execution)
# ==========================================

if __name__ == "__main__":
    # 1. Logger Setup
    logger.setup_logger()

    # 2. Initialize Infrastructure
    connection = database.initialize_infrastructure()

    # 3. Crawler Execution
    if connection:
        crawler.collect_news(connection)
        connection.close()
        logging.info(">>> Database connection closed. Program finished.")