# logger.py
import logging
import sys
import os
from logging.handlers import TimedRotatingFileHandler
import config  # 설정 가져오기

# ==========================================
# 2. 로깅 설정 (Logging Configuration)
# ==========================================
def setup_logger():
    # 인자가 있으면 그걸 쓰고, 없으면 기본값 사용
    if len(sys.argv) > 1:
        log_file_path = sys.argv[1]
    else:
        # 실행되는 파일 위치를 기준으로 경로 설정 (안전빵)
        # 주의: 이 함수를 호출하는 시점의 경로를 기준으로 함
        base_dir = os.getcwd() 
        log_dir = os.path.join(base_dir, config.DEFAULT_LOG_DIR)
        log_file_path = os.path.join(log_dir, config.DEFAULT_LOG_FILE)

    # [핵심 Fix] 로그 파일이 위치할 '폴더'가 없으면 무조건 생성
    log_dir_path = os.path.dirname(log_file_path)
    if not os.path.exists(log_dir_path):
        os.makedirs(log_dir_path, exist_ok=True)
        print(f">>> Log directory created: {log_dir_path}") # 디버깅용 출력

    # 로거 생성
    logger = logging.getLogger()
    
    # 이미 핸들러가 설정되어 있다면 중복 추가 방지
    if logger.hasHandlers():
        return logger
        
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
    # 로그 포맷: [레벨] 날짜 - 메시지 형식
    formatter = logging.Formatter('[%(levelname)s] %(asctime)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S')
    file_handler.setFormatter(formatter)
    logger.addHandler(file_handler)

    # 콘솔 출력용 핸들러 추가 (선택 사항)
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)
    
    return logger