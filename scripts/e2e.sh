#!/usr/bin/env bash
# End-to-end smoke test against the running docker compose stack.
# Exercises: health, profile, goal creation + validation, plan versioning,
# climbing log, chat session storage and the SSE coach error path (no LLM
# configured in the smoke environment).
set -euo pipefail
cd "$(dirname "$0")/.."

API="http://localhost:3000/api"
failures=0

check() {
  local description=$1
  local expected=$2
  local actual=$3
  if [ "$expected" = "$actual" ]; then
    echo "  ok: $description"
  else
    echo "  FAIL: $description (expected $expected, got $actual)"
    failures=$((failures + 1))
  fi
}

echo "vcoach e2e smoke"

echo "health"
check "healthz" "200" "$(curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/healthz)"
check "spa served" "200" "$(curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/)"
check "ping (resolver + db)" "200" "$(curl -s -o /dev/null -w '%{http_code}' $API/ping)"

echo "profile"
check "profile basics update" "200" "$(curl -s -o /dev/null -w '%{http_code}' -X PUT $API/profile \
  -H 'content-type: application/json' \
  -d '{"birthYear":1990,"sex":"male","weightKg":72.5,"runningExperience":"intermediate","climbingExperience":"advanced"}')"
check "profile read has derived" "yes" "$(curl -s $API/profile | python3 -c 'import sys,json; print("yes" if "derived" in json.load(sys.stdin) else "no")')"

echo "goals"
check "invalid goal rejected" "400" "$(curl -s -o /dev/null -w '%{http_code}' -X POST $API/goals \
  -H 'content-type: application/json' \
  -d '{"activityType":"boulder","targetMetric":"grade","targetValue":"9Z"}')"
GOAL_ID=$(curl -s -X POST $API/goals -H 'content-type: application/json' \
  -d '{"activityType":"boulder","targetMetric":"grade","targetValue":"7C","isPrimary":true}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')
check "valid goal created" "36" "${#GOAL_ID}"
check "goal progress computes" "200" "$(curl -s -o /dev/null -w '%{http_code}' $API/goals/$GOAL_ID/progress)"

echo "plan"
PLAN_BODY='{"title":"smoke plan","weeks":[{"weekNumber":1,"startDate":"2026-11-23","focus":"base","sessions":[{"sport":"boulder","sessionType":"limit bouldering","plannedDate":"2026-11-24","targetIntensity":"RPE 9"}]}]}'
PLAN1_VERSION=$(curl -s -X POST $API/plans -H 'content-type: application/json' -d "$PLAN_BODY" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["version"])')
PLAN2_VERSION=$(curl -s -X POST $API/plans -H 'content-type: application/json' -d "$PLAN_BODY" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["version"])')
check "plan version increments" "$((PLAN1_VERSION + 1))" "$PLAN2_VERSION"
check "plan list present" "200" "$(curl -s -o /dev/null -w '%{http_code}' $API/plans)"

echo "climbing log"
check "climbing session logged" "201" "$(curl -s -o /dev/null -w '%{http_code}' -X POST $API/climbing-sessions \
  -H 'content-type: application/json' \
  -d '{"sessionDate":"2026-11-22","venue":"smoke gym","discipline":"boulder","hardestSent":"6B","hardestAttempted":"6B+","attempts":10,"rpe":7}')"
check "climbing log listed" "200" "$(curl -s -o /dev/null -w '%{http_code}' $API/climbing-sessions)"

echo "chat"
SESSION_ID=$(curl -s -X POST $API/chat-sessions -H 'content-type: application/json' -d '{"title":"smoke"}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')
check "chat session created" "36" "${#SESSION_ID}"
check "user message persisted" "1" "$(curl -s -N -X POST $API/chat-sessions/$SESSION_ID/messages \
  -H 'content-type: application/json' -d '{"content":"smoke hello"}' >/dev/null; \
  curl -s $API/chat-sessions/$SESSION_ID/messages | python3 -c 'import sys,json; print(len(json.load(sys.stdin)))')"

echo "sync"
check "sync status present" "200" "$(curl -s -o /dev/null -w '%{http_code}' $API/sync/status)"

echo
if [ "$failures" -gt 0 ]; then
  echo "FAILED: $failures check(s) failed"
  exit 1
fi
echo "All checks passed."
