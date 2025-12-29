#!/bin/bash
set -e

echo "Running StinkyKitty CMS Enhancements Integration Tests"
echo "======================================================"

# Start server in background
./stinky server start &
SERVER_PID=$!
sleep 2

# Test 1: Site Settings - GA and Copyright
echo "Test 1: Checking site settings page..."
curl -s http://localhost:8080/admin/settings | grep -q "google_analytics_id" && echo "✓ GA field present" || echo "✗ GA field missing"
curl -s http://localhost:8080/admin/settings | grep -q "copyright_text" && echo "✓ Copyright field present" || echo "✗ Copyright field missing"

# Test 2: Public page header
echo "Test 2: Checking public page header..."
curl -s http://localhost:8080/ | grep -q "site-header" && echo "✓ Header present" || echo "✗ Header missing"
curl -s http://localhost:8080/ | grep -q "site-header-login" && echo "✓ Login button present" || echo "✗ Login button missing"

# Test 3: User management page
echo "Test 3: Checking user management..."
curl -s http://localhost:8080/admin/users | grep -q "User Management" && echo "✓ User management page accessible" || echo "✗ User management page missing"

# Test 4: Column block creation
echo "Test 4: Testing columns block..."
# This would require authenticated session - placeholder
echo "⊘ Column block test requires authentication"

# Cleanup
kill $SERVER_PID
echo ""
echo "Integration tests complete!"
