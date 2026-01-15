#!/bin/bash
# Test LinkedIn Messaging API Endpoints
# Usage: ./test_messaging.sh

set -e

CREDS_FILE="$HOME/.config/lnk/credentials.json"

if [ ! -f "$CREDS_FILE" ]; then
    echo "Error: No credentials found at $CREDS_FILE"
    echo "Run: lnk auth login --li-at YOUR_LI_AT --jsessionid YOUR_JSESSIONID"
    exit 1
fi

COOKIES=$(cat "$CREDS_FILE" | jq -r '"li_at=\(.li_at); JSESSIONID=\(.jsessionid)"')
CSRF=$(cat "$CREDS_FILE" | jq -r '.csrf_token')

echo "Testing LinkedIn Messaging API endpoints..."
echo "============================================"
echo ""

# Test 1: Verify auth works first
echo "1. Testing authentication (profile fetch)..."
PROFILE_RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/identity/dash/profiles?q=memberIdentity&memberIdentity=me&decorationId=com.linkedin.voyager.dash.deco.identity.profile.WebTopCardCore-19')

HTTP_CODE=$(echo "$PROFILE_RESULT" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✅ Auth OK (HTTP $HTTP_CODE)"
else
    echo "   ❌ Auth FAILED (HTTP $HTTP_CODE)"
    echo "   Please re-authenticate with fresh cookies"
    exit 1
fi
echo ""

# Test 2: New Dash Messaging Conversations
echo "2. Testing voyagerMessagingDashConversations..."
RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/voyagerMessagingDashConversations?decorationId=com.linkedin.voyager.dash.deco.messaging.FullConversation-46&count=10&q=syncToken')

HTTP_CODE=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | head -n -1)
INCLUDED_COUNT=$(echo "$BODY" | jq '.included | length' 2>/dev/null || echo "0")
echo "   HTTP: $HTTP_CODE, Included items: $INCLUDED_COUNT"
if [ "$HTTP_CODE" = "200" ] && [ "$INCLUDED_COUNT" != "0" ]; then
    echo "   ✅ This endpoint works!"
    echo "$BODY" | jq '.included[0]."$type"' 2>/dev/null || true
fi
echo ""

# Test 3: Messaging GraphQL
echo "3. Testing voyagerMessagingGraphQL..."
RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/voyagerMessagingGraphQL/graphql?queryId=messengerConversations.b82e44e85e0e8d228d5bb0e67d1c5c79&variables=(count:10)')

HTTP_CODE=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | head -n -1)
echo "   HTTP: $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    INCLUDED_COUNT=$(echo "$BODY" | jq '.included | length' 2>/dev/null || echo "0")
    echo "   Included items: $INCLUDED_COUNT"
    if [ "$INCLUDED_COUNT" != "0" ]; then
        echo "   ✅ This endpoint works!"
    fi
fi
echo ""

# Test 4: Legacy messaging
echo "4. Testing legacy /messaging/conversations..."
RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/messaging/conversations?keyVersion=LEGACY_INBOX')

HTTP_CODE=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | head -n -1)
echo "   HTTP: $HTTP_CODE"
STATUS=$(echo "$BODY" | jq '.data.status' 2>/dev/null || echo "null")
if [ "$STATUS" = "500" ]; then
    echo "   ❌ Endpoint returns internal status 500 (deprecated)"
elif [ "$HTTP_CODE" = "200" ]; then
    INCLUDED_COUNT=$(echo "$BODY" | jq '.included | length' 2>/dev/null || echo "0")
    echo "   Included items: $INCLUDED_COUNT"
    if [ "$INCLUDED_COUNT" != "0" ]; then
        echo "   ✅ This endpoint works!"
    fi
fi
echo ""

# Test 5: Messaging Threads
echo "5. Testing voyagerMessagingDashMessagingThreads..."
RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/voyagerMessagingDashMessagingThreads?decorationId=com.linkedin.voyager.dash.deco.messaging.Thread-7&count=10&q=inboxThreads')

HTTP_CODE=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | head -n -1)
echo "   HTTP: $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    INCLUDED_COUNT=$(echo "$BODY" | jq '.included | length' 2>/dev/null || echo "0")
    echo "   Included items: $INCLUDED_COUNT"
    if [ "$INCLUDED_COUNT" != "0" ]; then
        echo "   ✅ This endpoint works!"
    fi
fi
echo ""

# Test 6: Messaging Inbox
echo "6. Testing messaging inbox metadata..."
RESULT=$(curl -s -w "\n%{http_code}" \
  -H "Cookie: $COOKIES" \
  -H "Csrf-Token: $CSRF" \
  -H "X-Restli-Protocol-Version: 2.0.0" \
  -H "Accept: application/vnd.linkedin.normalized+json+2.1" \
  'https://www.linkedin.com/voyager/api/voyagerMessagingDashInbox')

HTTP_CODE=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | head -n -1)
echo "   HTTP: $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    echo "   Response: $(echo "$BODY" | jq -c '.data' 2>/dev/null | head -c 200)"
fi
echo ""

echo "============================================"
echo "Done. If all endpoints fail, LinkedIn may have"
echo "changed their API or restricted your account."
echo ""
echo "To get fresh cookies:"
echo "1. Open LinkedIn in your browser"
echo "2. Open Developer Tools (F12) > Application > Cookies"
echo "3. Copy li_at and JSESSIONID values"
echo "4. Run: lnk auth login --li-at VALUE --jsessionid VALUE"
