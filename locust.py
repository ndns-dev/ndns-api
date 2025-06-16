# load_test_patterns.py
import json
import time
from locust import HttpUser, task, between, events
import logging
import statistics
from collections import defaultdict
import hashlib
import glob
import os

# 로깅 설정
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# 테스트 시작 시간과 응답 데이터를 저장할 전역 변수
test_start_time = None
response_times = []
total_requests = 0
response_patterns = defaultdict(lambda: {
    "count": 0,
    "first_seen": None,
    "last_seen": None,
    "sample": None,
    "response_times": [],
    "errors": []
})

def get_response_hash(response_data):
    """응답 데이터의 해시값을 생성"""
    try:
        # 정렬된 JSON 문자열로 변환하여 일관된 해시 생성
        json_str = json.dumps(response_data, sort_keys=True)
        return hashlib.md5(json_str.encode()).hexdigest()[:8]
    except Exception as e:
        return f"error_{str(e)[:20]}"

def get_next_file_number():
    """다음 파일 번호를 찾음"""
    files = glob.glob("performance_summary*.json")
    if not files:
        return 1
    
    numbers = []
    for file in files:
        try:
            num = int(file.replace("performance_summary", "").replace(".json", ""))
            numbers.append(num)
        except ValueError:
            continue
    
    return max(numbers) + 1 if numbers else 1

class ApiUser(HttpUser):
    # 테스트할 기본 URL (k6의 BASE_URL과 동일)
    # host = "http://localhost:8085"
    # host = "https://route.ndns.site"
    host = "http://182.217.147.28:8085"
    # OCR 처리 시간을 고려하여 사용자 대기 시간을 12-15초로 설정
    wait_time = between(12, 15)

    SEARCH_QUERY = "병점 맛집"

    def on_start(self):
        """사용자 세션이 시작될 때 호출"""
        pass

    def on_stop(self):
        """사용자 세션이 종료될 때 호출"""
        pass

    @task
    def search_api(self):
        global total_requests
        total_requests += 1
        start_time = time.time()
        
        with self.client.get(
            f"/api/v1/search?query={self.SEARCH_QUERY}&limit=10&offset=0",
            name="/api/v1/search",
            catch_response=True
        ) as response:
            response_time = time.time() - start_time
            response_times.append(response_time)

            try:
                if response.status_code == 200:
                    response_data = response.json()
                    pattern_hash = get_response_hash(response_data)
                else:
                    # 에러 응답도 패턴으로 처리
                    pattern_hash = f"error_{response.status_code}"
                    response_data = {"error": f"HTTP {response.status_code}: {response.text}"}
            except json.JSONDecodeError as e:
                pattern_hash = "error_json_decode"
                response_data = {"error": f"JSON 디코딩 에러: {str(e)}"}
            except Exception as e:
                pattern_hash = f"error_unexpected_{type(e).__name__}"
                response_data = {"error": f"예상치 못한 에러: {str(e)}"}

            # 모든 요청에 대해 패턴 정보 업데이트
            pattern = response_patterns[pattern_hash]
            if pattern["count"] == 0:
                pattern["first_seen"] = time.time()
                pattern["sample"] = response_data
                logger.info(f"\n🆕 새로운 응답 패턴 발견! (패턴: {pattern_hash})")
                
            pattern["count"] += 1
            pattern["last_seen"] = time.time()
            pattern["response_times"].append(response_time)

            # OCR 처리 시간이 너무 길 경우 (15초 초과) 실패로 처리
            if response_time > 15:
                error_msg = f"응답 시간 초과: {response_time:.2f}초"
                pattern["errors"].append(error_msg)
                response.failure(error_msg)
            elif response.status_code != 200:
                error_msg = f"HTTP {response.status_code}: {response.text}"
                pattern["errors"].append(error_msg)
                response.failure(error_msg)
            else:
                response.success()

# --- Locust 이벤트 핸들러 ---
def on_request_completion(request_type, name, response_time, response_length, response, exception, **kwargs):
    # 이벤트 핸들러에서는 패턴 카운팅을 하지 않고 로깅만 수행
    if name == "/api/v1/search" and response.status_code == 200 and exception is None:
        try:
            response_body = response.json()
            logger.debug(f"Response received: {json.dumps(response_body, indent=2, ensure_ascii=False)}")
        except json.JSONDecodeError:
            logger.error(f"Failed to decode JSON in request completion handler: {response.text}")
        except Exception as e:
            logger.error(f"Error in request completion handler: {e}")

# init 이벤트 핸들러는 Locust 환경이 초기화된 후에 호출됩니다.
# 이 안에서 on_request_completion 함수를 environment.events.request에 등록합니다.
@events.init.add_listener
def on_locust_init(environment, **kwargs):
    # request_success 대신 request 이벤트를 사용합니다.
    environment.events.request.add_listener(on_request_completion)

# 테스트 시작 시 호출되는 이벤트 핸들러
@events.test_start.add_listener
def on_test_start(**kwargs):
    global test_start_time, response_times, response_patterns, total_requests
    test_start_time = time.time()
    response_times = []
    total_requests = 0
    response_patterns.clear()
    logger.info("🚀 부하 테스트 시작")

# 테스트 종료 시 호출되는 이벤트 핸들러
@events.test_stop.add_listener
def on_test_stop(**kwargs):
    if not response_times:
        logger.info("수집된 테스트 데이터가 없습니다")
        return

    # 기본 성능 통계 계산
    avg_response_time = statistics.mean(response_times)
    median_response_time = statistics.median(response_times)
    p95_response_time = statistics.quantiles(response_times, n=20)[18]  # 95th percentile
    min_response_time = min(response_times)
    max_response_time = max(response_times)

    # 응답 패턴 분석
    patterns_summary = []
    pattern_total_count = sum(pattern["count"] for pattern in response_patterns.values())
    
    for pattern_id, data in response_patterns.items():
        pattern_avg_time = statistics.mean(data["response_times"]) if data["response_times"] else 0
        
        patterns_summary.append({
            "pattern_id": pattern_id,
            "count": data["count"],
            "percentage": f"{(data['count'] / total_requests * 100):.1f}%",
            "avg_response_time": f"{pattern_avg_time:.2f}s",
            "first_seen": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(data["first_seen"])),
            "last_seen": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(data["last_seen"])),
            "sample": data["sample"],
            "error_count": len(data["errors"]),
            "recent_errors": data["errors"][-5:] if data["errors"] else []  # 최근 5개 에러만 저장
        })

    # 결과 요약
    summary = {
        "performance_metrics": {
            "average_response_time": f"{avg_response_time:.2f}s",
            "median_response_time": f"{median_response_time:.2f}s",
            "p95_response_time": f"{p95_response_time:.2f}s",
            "min_response_time": f"{min_response_time:.2f}s",
            "max_response_time": f"{max_response_time:.2f}s",
            "total_requests": total_requests,
            "pattern_total_count": pattern_total_count
        },
        "test_duration": {
            "start_time": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(test_start_time)),
            "end_time": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(time.time())),
            "duration_seconds": f"{time.time() - test_start_time:.2f}"
        },
        "response_patterns": patterns_summary
    }

    # 결과 출력
    logger.info("\n📊 성능 테스트 요약")
    logger.info("===================")
    logger.info(f"총 요청 수: {total_requests}")
    logger.info(f"패턴별 응답 합계: {pattern_total_count}")
    logger.info(f"평균 응답 시간: {summary['performance_metrics']['average_response_time']}")
    logger.info(f"중간값 응답 시간: {summary['performance_metrics']['median_response_time']}")
    logger.info(f"95번째 백분위 응답 시간: {summary['performance_metrics']['p95_response_time']}")
    logger.info(f"최소 응답 시간: {summary['performance_metrics']['min_response_time']}")
    logger.info(f"최대 응답 시간: {summary['performance_metrics']['max_response_time']}")
    
    logger.info("\n📋 응답 패턴 분석")
    logger.info("===================")
    for pattern in patterns_summary:
        pattern_type = "성공" if not pattern["pattern_id"].startswith("error") else "실패"
        logger.info(f"\n패턴 {pattern['pattern_id']} ({pattern_type}):")
        logger.info(f"  발생 횟수: {pattern['count']} ({pattern['percentage']})")
        logger.info(f"  평균 응답 시간: {pattern['avg_response_time']}")
        if pattern["error_count"] > 0:
            logger.info(f"  에러 횟수: {pattern['error_count']}")
            logger.info("  최근 에러:")
            for error in pattern["recent_errors"]:
                logger.info(f"    - {error}")

    # 다음 파일 번호 가져오기
    next_num = get_next_file_number()
    filename = f"performance_summary{next_num}.json"

    # JSON 파일로 저장
    with open(filename, "w", encoding="utf-8") as f:
        json.dump(summary, f, indent=2, ensure_ascii=False)
    
    logger.info(f"\n✅ 성능 테스트 결과가 {filename} 파일에 저장되었습니다")
