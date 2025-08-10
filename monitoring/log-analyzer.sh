#!/bin/bash
# Inspector Gadget OS Log Analyzer
# Provides real-time analysis and alerting for the logging system

set -e

LOG_FILE="${1:-/var/log/inspector-gadget-os/server.log}"
ALERT_THRESHOLD_AUTH_FAILURES=5
ALERT_THRESHOLD_RESPONSE_TIME=5000
ALERT_WINDOW_MINUTES=5

if [[ ! -f "$LOG_FILE" ]]; then
    echo "❌ Log file not found: $LOG_FILE"
    echo "Usage: $0 [log_file_path]"
    exit 1
fi

echo "🔍 Inspector Gadget OS Log Analyzer"
echo "=================================="
echo "Log file: $LOG_FILE"
echo "Analysis window: Last $ALERT_WINDOW_MINUTES minutes"
echo ""

# Get timestamp for analysis window
WINDOW_START=$(date -d "$ALERT_WINDOW_MINUTES minutes ago" '+%Y-%m-%dT%H:%M')

# Function to analyze authentication failures
analyze_auth_failures() {
    echo "🔐 Authentication Analysis"
    echo "-------------------------"
    
    local failures=$(grep "\"level\":\"error\"" "$LOG_FILE" | \
                    grep "\"component\":\"auth\"" | \
                    grep "$WINDOW_START" | \
                    jq -r '.source_ip' 2>/dev/null | \
                    sort | uniq -c | \
                    awk -v threshold="$ALERT_THRESHOLD_AUTH_FAILURES" '$1 >= threshold')
    
    if [[ -n "$failures" ]]; then
        echo "🚨 ALERT: High authentication failure rate detected!"
        echo "$failures"
        echo ""
    else
        echo "✅ No authentication anomalies detected"
        echo ""
    fi
    
    # Top failed authentication sources
    local top_failures=$(grep "\"level\":\"error\"" "$LOG_FILE" | \
                        grep "\"component\":\"auth\"" | \
                        jq -r '.source_ip' 2>/dev/null | \
                        sort | uniq -c | sort -nr | head -5)
    
    if [[ -n "$top_failures" ]]; then
        echo "Top authentication failure sources (all time):"
        echo "$top_failures"
        echo ""
    fi
}

# Function to analyze performance
analyze_performance() {
    echo "⚡ Performance Analysis"
    echo "---------------------"
    
    # Slow requests in the time window
    local slow_requests=$(grep "\"duration_ms\"" "$LOG_FILE" | \
                         grep "$WINDOW_START" | \
                         jq -r "select(.duration_ms > $ALERT_THRESHOLD_RESPONSE_TIME) | \"\(.path) \(.duration_ms)ms\"" 2>/dev/null)
    
    if [[ -n "$slow_requests" ]]; then
        echo "🚨 ALERT: Slow requests detected (>${ALERT_THRESHOLD_RESPONSE_TIME}ms)!"
        echo "$slow_requests"
        echo ""
    else
        echo "✅ No performance anomalies detected"
        echo ""
    fi
    
    # Average response times by endpoint
    local avg_times=$(grep "\"duration_ms\"" "$LOG_FILE" | \
                     jq -r '"\(.path) \(.duration_ms)"' 2>/dev/null | \
                     awk '{path[$1] += $2; count[$1]++} END {
                         for (p in path) printf "%-30s %.2fms\n", p, path[p]/count[p]
                     }' | sort -k2 -nr | head -10)
    
    if [[ -n "$avg_times" ]]; then
        echo "Slowest endpoints (average response time):"
        echo "$avg_times"
        echo ""
    fi
}

# Function to analyze security events
analyze_security() {
    echo "🛡️ Security Analysis"
    echo "-------------------"
    
    # Path traversal attempts
    local path_traversal=$(grep "\"level\":\"error\"" "$LOG_FILE" | \
                          grep "path_traversal" | \
                          grep "$WINDOW_START" | \
                          jq -r '"\(.timestamp) \(.user_id // \"anonymous\") \(.attempted_path)"' 2>/dev/null)
    
    if [[ -n "$path_traversal" ]]; then
        echo "🚨 ALERT: Path traversal attempts detected!"
        echo "$path_traversal"
        echo ""
    fi
    
    # RBAC violations
    local rbac_violations=$(grep "\"level\":\"warn\"" "$LOG_FILE" | \
                           grep "\"component\":\"rbac\"" | \
                           grep "access denied" | \
                           grep "$WINDOW_START" | \
                           jq -r '"\(.user_id) -> \(.resource) (\(.action))"' 2>/dev/null | \
                           sort | uniq -c)
    
    if [[ -n "$rbac_violations" ]]; then
        echo "⚠️  Recent RBAC access denials:"
        echo "$rbac_violations"
        echo ""
    fi
    
    # Suspicious file access patterns
    local suspicious_files=$(grep "\"component\":\"safefs\"" "$LOG_FILE" | \
                            grep -E '(/etc|/root|\.ssh|\.key|\.pem)' | \
                            jq -r '"\(.user_id) accessed \(.path)"' 2>/dev/null | \
                            sort | uniq -c | sort -nr | head -5)
    
    if [[ -n "$suspicious_files" ]]; then
        echo "👁️  Sensitive file access patterns:"
        echo "$suspicious_files"
        echo ""
    fi
}

# Function to analyze gadget usage
analyze_gadgets() {
    echo "🎯 Gadget Usage Analysis"
    echo "-----------------------"
    
    # Most used gadgets
    local popular_gadgets=$(grep "\"component\":\"gadgets\"" "$LOG_FILE" | \
                           grep "execution started" | \
                           jq -r '.gadget_name' 2>/dev/null | \
                           sort | uniq -c | sort -nr | head -10)
    
    if [[ -n "$popular_gadgets" ]]; then
        echo "Most popular gadgets:"
        echo "$popular_gadgets"
        echo ""
    fi
    
    # Failed gadget executions
    local failed_gadgets=$(grep "\"component\":\"gadgets\"" "$LOG_FILE" | \
                          grep "\"level\":\"error\"" | \
                          grep "$WINDOW_START" | \
                          jq -r '"\(.gadget_name) - \(.error)"' 2>/dev/null)
    
    if [[ -n "$failed_gadgets" ]]; then
        echo "⚠️  Recent gadget failures:"
        echo "$failed_gadgets"
        echo ""
    fi
}

# Function to analyze user activity
analyze_users() {
    echo "👥 User Activity Analysis"
    echo "------------------------"
    
    # Most active users
    local active_users=$(grep "\"user_id\"" "$LOG_FILE" | \
                        jq -r '.user_id' 2>/dev/null | \
                        grep -v "null" | \
                        sort | uniq -c | sort -nr | head -10)
    
    if [[ -n "$active_users" ]]; then
        echo "Most active users:"
        echo "$active_users"
        echo ""
    fi
    
    # Recent new users
    local new_users=$(grep "\"component\":\"auth\"" "$LOG_FILE" | \
                     grep "user registered" | \
                     grep "$WINDOW_START" | \
                     jq -r '.user_id' 2>/dev/null)
    
    if [[ -n "$new_users" ]]; then
        echo "New user registrations (last $ALERT_WINDOW_MINUTES minutes):"
        echo "$new_users"
        echo ""
    fi
}

# Function to show system health summary
show_health_summary() {
    echo "📊 System Health Summary"
    echo "======================="
    
    local total_requests=$(grep "\"duration_ms\"" "$LOG_FILE" | wc -l)
    local error_count=$(grep "\"level\":\"error\"" "$LOG_FILE" | wc -l)
    local warning_count=$(grep "\"level\":\"warn\"" "$LOG_FILE" | wc -l)
    
    if [[ $total_requests -gt 0 ]]; then
        local error_rate=$(echo "scale=2; $error_count * 100 / $total_requests" | bc -l 2>/dev/null || echo "0")
        echo "Total requests: $total_requests"
        echo "Errors: $error_count (${error_rate}%)"
        echo "Warnings: $warning_count"
    else
        echo "No request data available"
    fi
    
    # Recent uptime from health checks
    local last_health=$(grep "health check" "$LOG_FILE" | tail -1 | jq -r '.uptime_seconds' 2>/dev/null)
    if [[ -n "$last_health" && "$last_health" != "null" ]]; then
        local uptime_hours=$(echo "scale=1; $last_health / 3600" | bc -l)
        echo "System uptime: ${uptime_hours} hours"
    fi
    
    echo ""
}

# Main analysis
main() {
    show_health_summary
    analyze_auth_failures
    analyze_performance
    analyze_security
    analyze_gadgets
    analyze_users
    
    echo "Analysis completed at $(date)"
    echo ""
    echo "💡 Tips:"
    echo "- Run with -f flag to monitor in real-time: watch -n 30 $0"
    echo "- Set custom thresholds with environment variables:"
    echo "  ALERT_THRESHOLD_AUTH_FAILURES=10 ALERT_THRESHOLD_RESPONSE_TIME=1000 $0"
    echo "- View raw logs: tail -f $LOG_FILE | jq '.'"
}

# Handle command line options
case "${2:-}" in
    "--watch")
        echo "📡 Monitoring mode - updating every 30 seconds..."
        watch -n 30 "$0" "$LOG_FILE"
        ;;
    "--alerts-only")
        # Only show alerts, no regular stats
        ALERT_THRESHOLD_AUTH_FAILURES=1
        analyze_auth_failures
        analyze_performance | grep "🚨"
        analyze_security | grep -E "(🚨|⚠️)"
        ;;
    *)
        main
        ;;
esac