#!/bin/bash
# File: test-api.sh
# Comprehensive API testing script

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="https://localhost"
INSECURE="-k"  # Use -k for self-signed certificates
VERBOSE=""     # Set to "-v" for verbose curl output

# Counters
TESTS_RUN=0
TESTS_PASSED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((TESTS_PASSED++))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_test() {
    ((TESTS_RUN++))
    echo -e "\n${BLUE}Test $TESTS_RUN:${NC} $1"
}

check_response() {
    local response="$1"
    local expected_status="$2"
    local test_name="$3"
    
    if echo "$response" | grep -q "\"success\":true" && [ "$expected_status" = "200" ]; then
        log_success "$test_name"
        return 0
    elif echo "$response" | grep -q "\"success\":false" && [ "$expected_status" != "200" ]; then
        log_success "$test_name (expected failure)"
        return 0
    else
        log_error "$test_name - Unexpected response: $response"
        return 1
    fi
}

# Check if API is running
check_api_status() {
    run_test "Checking API availability"
    
    if curl $INSECURE -s --connect-timeout 5 "$API_URL/health" > /dev/null 2>&1; then
        log_success "API is running and accessible"
    else
        log_error "API is not accessible at $API_URL"
        log_info "Make sure the services are running: docker-compose up -d"
        exit 1
    fi
}

# Test health endpoints
test_health_endpoints() {
    run_test "Testing basic health endpoint"
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/health")
    check_response "$response" "200" "Basic health check"
    
    run_test "Testing detailed health endpoint"
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/health/detailed")
    check_response "$response" "200" "Detailed health check"
}

# Test security headers
test_security_headers() {
    run_test "Testing security headers"
    headers=$(curl $INSECURE -s -I "$API_URL/health")
    
    if echo "$headers" | grep -q "X-Content-Type-Options: nosniff"; then
        log_success "X-Content-Type-Options header present"
    else
        log_error "Missing X-Content-Type-Options header"
    fi
    
    if echo "$headers" | grep -q "X-Frame-Options: DENY"; then
        log_success "X-Frame-Options header present"
    else
        log_error "Missing X-Frame-Options header"
    fi
    
    if echo "$headers" | grep -q "Strict-Transport-Security"; then
        log_success "HSTS header present"
    else
        log_error "Missing HSTS header"
    fi
}

# Test user registration
test_user_registration() {
    run_test "Testing user registration"
    
    # Generate unique test user
    timestamp=$(date +%s)
    test_username="testuser$timestamp"
    test_email="test$timestamp@example.com"
    test_password="TestPassword123!"
    
    response=$(curl $INSECURE $VERBOSE -s -X POST "$API_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$test_username\",
            \"email\": \"$test_email\",
            \"password\": \"$test_password\"
        }")
    
    if check_response "$response" "200" "User registration"; then
        # Store credentials for later tests
        export TEST_USERNAME="$test_username"
        export TEST_EMAIL="$test_email"
        export TEST_PASSWORD="$test_password"
        log_info "Created test user: $test_username"
    fi
}

# Test user authentication
test_authentication() {
    run_test "Testing user authentication"
    
    if [ -z "$TEST_USERNAME" ]; then
        log_warning "Skipping authentication test - no test user available"
        return
    fi
    
    response=$(curl $INSECURE $VERBOSE -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$TEST_USERNAME\",
            \"password\": \"$TEST_PASSWORD\"
        }")
    
    if check_response "$response" "200" "User authentication"; then
        # Extract token for protected endpoint tests
        export TEST_TOKEN=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$TEST_TOKEN" ]; then
            log_info "JWT token acquired successfully"
        else
            log_error "Failed to extract JWT token"
        fi
    fi
}

# Test protected endpoints
test_protected_endpoints() {
    run_test "Testing protected endpoints without token"
    
    # Should fail without authentication
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/api/v1/profile")
    if echo "$response" | grep -q "Authorization header required"; then
        log_success "Protected endpoint correctly rejects unauthenticated requests"
    else
        log_error "Protected endpoint should reject unauthenticated requests"
    fi
    
    if [ -z "$TEST_TOKEN" ]; then
        log_warning "Skipping authenticated protected endpoint tests - no token available"
        return
    fi
    
    run_test "Testing protected endpoints with valid token"
    
    # Test profile endpoint
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/api/v1/profile" \
        -H "Authorization: Bearer $TEST_TOKEN")
    check_response "$response" "200" "Get user profile"
    
    # Test users list endpoint
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/api/v1/users?page=1&limit=5" \
        -H "Authorization: Bearer $TEST_TOKEN")
    check_response "$response" "200" "Get users list"
    
    # Test protected example endpoint
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/api/v1/protected" \
        -H "Authorization: Bearer $TEST_TOKEN")
    check_response "$response" "200" "Protected example endpoint"
}

# Test input validation
test_input_validation() {
    run_test "Testing input validation"
    
    # Test registration with invalid data
    response=$(curl $INSECURE $VERBOSE -s -X POST "$API_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"ab\",
            \"email\": \"invalid-email\",
            \"password\": \"weak\"
        }")
    
    if echo "$response" | grep -q "validation failed"; then
        log_success "Input validation working correctly"
    else
        log_error "Input validation not working properly"
    fi
}

# Test rate limiting
test_rate_limiting() {
    run_test "Testing rate limiting"
    
    log_info "Making rapid requests to test rate limiting..."
    
    # Make rapid requests to trigger rate limit
    rate_limited=false
    for i in {1..15}; do
        response=$(curl $INSECURE -s -w "%{http_code}" "$API_URL/health")
        if echo "$response" | grep -q "429"; then
            rate_limited=true
            break
        fi
        sleep 0.1
    done
    
    if [ "$rate_limited" = true ]; then
        log_success "Rate limiting is working"
    else
        log_warning "Rate limiting might not be configured or limit is too high"
    fi
}

# Test database connectivity
test_database_stats() {
    run_test "Testing database statistics endpoint"
    
    if [ -z "$TEST_TOKEN" ]; then
        log_warning "Skipping database stats test - no token available"
        return
    fi
    
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/api/v1/admin/db-stats" \
        -H "Authorization: Bearer $TEST_TOKEN")
    check_response "$response" "200" "Database statistics"
}

# Test metrics endpoint
test_metrics() {
    run_test "Testing Prometheus metrics"
    
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/metrics")
    if echo "$response" | grep -q "go_info"; then
        log_success "Prometheus metrics available"
    else
        log_error "Prometheus metrics not available"
    fi
}

# Test error handling
test_error_handling() {
    run_test "Testing error handling"
    
    # Test 404
    response=$(curl $INSECURE $VERBOSE -s "$API_URL/nonexistent-endpoint")
    if echo "$response" | grep -q "404"; then
        log_success "404 error handling working"
    else
        log_warning "404 error handling might not be optimal"
    fi
    
    # Test malformed JSON
    response=$(curl $INSECURE $VERBOSE -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d "invalid-json")
    if echo "$response" | grep -q "Invalid request"; then
        log_success "Malformed JSON handling working"
    else
        log_warning "Malformed JSON handling might need improvement"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}=== Go API Production Readiness Test Suite ===${NC}\n"
    
    log_info "Testing API at: $API_URL"
    log_info "Using insecure mode for self-signed certificates"
    
    # Run all tests
    check_api_status
    test_health_endpoints
    test_security_headers
    test_user_registration
    test_authentication
    test_protected_endpoints
    test_input_validation
    test_rate_limiting
    test_database_stats
    test_metrics
    test_error_handling
    
    # Summary
    echo -e "\n${BLUE}=== Test Summary ===${NC}"
    echo "Tests run: $TESTS_RUN"
    echo "Tests passed: $TESTS_PASSED"
    
    if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
        echo -e "${GREEN}All tests passed! 🎉${NC}"
        echo -e "${GREEN}Your API is production-ready!${NC}"
        exit 0
    else
        failed=$((TESTS_RUN - TESTS_PASSED))
        echo -e "${YELLOW}$failed tests failed or had warnings${NC}"
        echo -e "${YELLOW}Review the output above for details${NC}"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --verbose, -v  Enable verbose curl output"
        echo "  --url URL      Use custom API URL (default: https://localhost)"
        echo ""
        echo "Example: $0 --verbose --url https://api.example.com"
        exit 0
        ;;
    --verbose|-v)
        VERBOSE="-v"
        ;;
    --url)
        API_URL="$2"
        shift
        ;;
esac

# Run the tests
main