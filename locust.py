# load_test_patterns.py
import json
import time
from locust import HttpUser, task, between, events
from collections import defaultdict
import logging

# 로깅 설정
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# 응답 패턴을 저장할 전역 딕셔너리 (Locust의 Master/Worker 모드에서 집계될 때 사용)
# defaultdict를 사용하여 새로운 패턴이 등장하면 자동으로 초기화되도록 설정
response_patterns = defaultdict(lambda: {
    "count": 0,
    "firstSeen": None,
    "lastSeen": None,
    "sampleResponse": {}
})

# 응답 JSON 본문을 일관된 문자열(해시)로 변환하는 함수
# JSON 키 순서가 중요하지 않다면, sort_keys=True를 사용하여 일관성 보장
def canonicalize_json(json_data):
    return json.dumps(json_data, sort_keys=True, ensure_ascii=False)

class ApiUser(HttpUser):
    # 테스트할 기본 URL (k6의 BASE_URL과 동일)
    host = "http://localhost:8085"
    # 각 요청 사이에 1초 대기 (k6의 sleep(1)과 유사)
    wait_time = between(1, 1)

    SEARCH_QUERY = "병점 맛집"

    @task
    def health_check(self):
        self.client.get("/health", name="/health") # name으로 요청을 그룹화하여 통계에 표시

    @task
    def search_api(self):
        # 검색 API 호출
        response = self.client.get(
            f"/api/v1/search?query={self.SEARCH_QUERY}&limit=10&offset=0",
            name="/api/v1/search" # name으로 요청을 그룹화하여 통계에 표시
        )

        if response.status_code == 200:
            try:
                # 응답 본문 파싱
                response_body = response.json()
                # 응답 본문을 정규화된 JSON 문자열로 변환 (패턴 식별용)
                pattern_hash = canonicalize_json(response_body)

                # 현재 시간 기록 (여기서는 직접적인 사용은 없지만, on_request_completion에서 사용될 수 있음)
                current_time = time.time()

                # 응답 패턴 집계는 on_request_completion 핸들러에서 이루어집니다.
                # 이 task 내부에서는 HTTP 요청 자체의 성공 여부만 확인합니다.

            except json.JSONDecodeError:
                logger.error(f"Failed to decode JSON from response: {response.text}")
            except Exception as e:
                logger.error(f"Error processing response: {e}")
        else:
            logger.error(f"Search API returned status {response.status_code}: {response.text}")

# --- Locust 이벤트 핸들러 ---
# on_request_completion 함수는 이제 일반 함수로 정의됩니다.
# request 이벤트에 이 함수를 등록하여 성공/실패 요청 모두 처리합니다.
def on_request_completion(request_type, name, response_time, response_length, response, exception, **kwargs):
    # 'request' 이벤트는 성공/실패 모두에 대해 발생하므로,
    # 성공한 검색 API 요청에 대해서만 패턴을 집계합니다.
    if name == "/api/v1/search" and response.status_code == 200 and exception is None:
        try:
            response_body = response.json()
            pattern_hash = canonicalize_json(response_body)
            
            # 응답 패턴 업데이트
            if response_patterns[pattern_hash]["count"] == 0:
                response_patterns[pattern_hash]["firstSeen"] = time.time()
                response_patterns[pattern_hash]["sampleResponse"] = response_body # 샘플 응답 저장 (너무 크면 문제)
                logger.info(f"\n🆕 New response pattern found! (Hash: {pattern_hash[:30]}...)")
                logger.info(f"Sample response: {json.dumps(response_body, indent=2, ensure_ascii=False)}")
                logger.info("-------------------------------------")
            
            response_patterns[pattern_hash]["count"] += 1
            response_patterns[pattern_hash]["lastSeen"] = time.time()

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


# 테스트가 종료될 때 실행되는 함수
@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    logger.info("\n📊 Test Summary for Response Patterns")
    logger.info("======================================")

    total_search_requests = 0
    for pattern_info in response_patterns.values():
        total_search_requests += pattern_info["count"]

    if not response_patterns:
        logger.info("No search API responses collected for pattern analysis.")
        return

    # 결과 요약 데이터 준비
    summary_data = {
        "query": ApiUser.SEARCH_QUERY,
        "total_search_requests": total_search_requests,
        "total_unique_patterns": len(response_patterns),
        "test_start_time": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(environment.start_time)),
        "test_end_time": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(time.time())),
        "patterns": []
    }

    # 각 패턴 정보 추가
    label_index = 0
    for pattern_hash, pattern_info in response_patterns.items():
        # 패턴 레이블은 단순히 순번으로 부여
        label_index += 1

        summary_data["patterns"].append({
            "label": f"Pattern {label_index}",
            "count": pattern_info["count"],
            "percentage": f"{((pattern_info['count'] / total_search_requests) * 100):.2f}%" if total_search_requests > 0 else "0.00%",
            "firstSeen": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(pattern_info["firstSeen"])) if pattern_info["firstSeen"] else None,
            "lastSeen": time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(pattern_info["lastSeen"])),
            "responseSample": pattern_info["sampleResponse"]
        })
        
        logger.info(f"\n🏷 Pattern {label_index}:")
        logger.info(f"   Count: {pattern_info['count']} times")
        logger.info(f"   Percentage: {((pattern_info['count'] / total_search_requests) * 100):.2f}%")
        logger.info(f"   First Seen: {time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(pattern_info['firstSeen'])) if pattern_info['firstSeen'] else 'N/A'}")
        logger.info(f"   Last Seen: {time.strftime('%Y-%m-%dT%H:%M:%S%Z', time.localtime(pattern_info['lastSeen']))}")
        logger.info(f"   Sample Response: {json.dumps(pattern_info['sampleResponse'], indent=2, ensure_ascii=False).splitlines()[0]}...")


    # 결과를 JSON 파일로 저장
    output_filename = "response_patterns_summary.json"
    with open(output_filename, "w", encoding="utf-8") as f:
        json.dump(summary_data, f, indent=2, ensure_ascii=False)
    
    logger.info(f"\n✅ Response patterns summary saved to {output_filename}")
