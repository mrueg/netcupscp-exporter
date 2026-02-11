#!/bin/bash

# Configuration
CLIENT_ID="scp"
DEVICE_AUTH_URL="https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/auth/device"
TOKEN_URL="https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token"

echo "Step 1: Requesting device code..."
RESPONSE=$(curl -s -X POST "$DEVICE_AUTH_URL" -d "client_id=$CLIENT_ID" -d "scope=offline_access openid")

DEVICE_CODE=$(echo "$RESPONSE" | grep -oP '"device_code":"\K[^"]+')
USER_CODE=$(echo "$RESPONSE" | grep -oP '"user_code":"\K[^"]+')
VERIFY_URL=$(echo "$RESPONSE" | grep -oP '"verification_uri_complete":"\K[^"]+')
INTERVAL=$(echo "$RESPONSE" | grep -oP '"interval":\K[0-9]+')

if [ -z "$DEVICE_CODE" ]; then
    echo "Error: Could not obtain device code."
    echo "Response: $RESPONSE"
    exit 1
fi

echo "----------------------------------------------------------------"
echo "Please visit the following URL in your browser and log in:"
echo ""
echo "$VERIFY_URL"
echo ""
echo "User Code: $USER_CODE"
echo "----------------------------------------------------------------"
echo "Waiting for authentication..."

while true; do
    sleep "$INTERVAL"
    
    TOKEN_RESPONSE=$(curl -s -X POST "$TOKEN_URL" -d "grant_type=urn:ietf:params:oauth:grant-type:device_code"  -d "device_code=$DEVICE_CODE" -d "client_id=$CLIENT_ID")
    
    # Check if we got an error
    ERROR=$(echo "$TOKEN_RESPONSE" | grep -oP '"error":"\K[^"]+')
    
    if [ -z "$ERROR" ]; then
        REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -oP '"refresh_token":"\K[^"]+')
        if [ -n "$REFRESH_TOKEN" ]; then
            echo ""
            echo "Success! Your Refresh Token is:"
            echo ""
            echo "$REFRESH_TOKEN"
            echo ""
            echo "You can now use this token with the exporter:"
            echo "./netcupscp-exporter --refresh-token $REFRESH_TOKEN"
            exit 0
        fi
    elif [ "$ERROR" != "authorization_pending" ]; then
        echo "Error: $ERROR"
        echo "Response: $TOKEN_RESPONSE"
        exit 1
    fi
    
    echo -n "."
done
