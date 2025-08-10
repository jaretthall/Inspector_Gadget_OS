// Package integration provides integration between O-LLaMA and the gadget framework
package integration

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"inspector-gadget-os/o-llama/internal/auth"
    "inspector-gadget-os/o-llama/internal/logging"
	"inspector-gadget-os/o-llama/internal/rbac"
)

// GadgetIntegration provides secure integration with the gadget framework
type GadgetIntegration struct {
	gadgetBinaryPath string
	rbacMiddleware   *rbac.RBACMiddleware
}

// GadgetExecuteRequest represents a request to execute a gadget
type GadgetExecuteRequest struct {
	GadgetName string   `json:"gadget_name" binding:"required"`
	Args       []string `json:"args,omitempty"`
	Flags      []string `json:"flags,omitempty"`
}

// GadgetExecuteResponse represents the response from gadget execution
type GadgetExecuteResponse struct {
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exit_code"`
	GadgetName string `json:"gadget_name"`
}

// GadgetInfo represents information about a gadget
type GadgetInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`
}

// GadgetListResponse represents the response from listing gadgets
type GadgetListResponse struct {
	Gadgets []GadgetInfo `json:"gadgets"`
	Count   int          `json:"count"`
}

// NewGadgetIntegration creates a new gadget integration
func NewGadgetIntegration(gadgetBinaryPath string, rbacMiddleware *rbac.RBACMiddleware) *GadgetIntegration {
	return &GadgetIntegration{
		gadgetBinaryPath: gadgetBinaryPath,
		rbacMiddleware:   rbacMiddleware,
	}
}

// RegisterRoutes registers gadget integration routes
func (gi *GadgetIntegration) RegisterRoutes(router *gin.RouterGroup) {
	gadgets := router.Group("/gadgets")
	
	// List available gadgets (requires user role)
	gadgets.GET("", gi.rbacMiddleware.UserOrAdmin(), gi.ListGadgets)
	
	// Get gadget information (requires user role)
	gadgets.GET("/:name/info", gi.rbacMiddleware.UserOrAdmin(), gi.GetGadgetInfo)
	
	// Execute gadget (requires appropriate permissions)
	gadgets.POST("/:name/execute", gi.rbacMiddleware.RequirePermission("gadgets", "execute"), gi.ExecuteGadget)
	
	// System gadgets require admin permissions
	gadgets.POST("/sysinfo/execute", gi.rbacMiddleware.AdminOnly(), gi.ExecuteSystemGadget)
}

// ListGadgets returns all available gadgets
func (gi *GadgetIntegration) ListGadgets(c *gin.Context) {
    cmd := exec.Command(gi.gadgetBinaryPath, "list")
	output, err := cmd.Output()
	if err != nil {
        logging.L().Errorw("gadget.list.error", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list gadgets",
			"details": err.Error(),
		})
		return
	}

	// Parse gadget list output (this would depend on the actual output format)
	gadgets := gi.parseGadgetList(string(output))
	
    response := GadgetListResponse{
		Gadgets: gadgets,
		Count:   len(gadgets),
	}
	
    logging.L().Infow("gadget.list.ok", "count", response.Count)
	c.JSON(http.StatusOK, response)
}

// GetGadgetInfo returns information about a specific gadget
func (gi *GadgetIntegration) GetGadgetInfo(c *gin.Context) {
	gadgetName := c.Param("name")
	
	// Validate gadget name for security
	if !gi.isValidGadgetName(gadgetName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gadget name"})
		return
	}
	
    cmd := exec.Command(gi.gadgetBinaryPath, "info", gadgetName)
	output, err := cmd.Output()
	if err != nil {
        logging.L().Warnw("gadget.info.error", "gadget_name", gadgetName, "error", err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Gadget not found or failed to get info",
			"gadget": gadgetName,
		})
		return
	}
	
	// Parse gadget info (this would depend on the actual output format)
    info := gi.parseGadgetInfo(string(output), gadgetName)
    logging.L().Infow("gadget.info.ok", "gadget_name", gadgetName)
	c.JSON(http.StatusOK, info)
}

// ExecuteGadget executes a gadget with security checks
func (gi *GadgetIntegration) ExecuteGadget(c *gin.Context) {
	gadgetName := c.Param("name")
	
	// Validate gadget name
	if !gi.isValidGadgetName(gadgetName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gadget name"})
		return
	}
	
    var req GadgetExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user info for audit logging
	claims, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	// Security check: prevent system gadgets from being executed without admin rights
	if gi.isSystemGadget(gadgetName) && !gi.rbacMiddleware.CheckPermission(c, "system", "manage") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "System gadgets require admin permissions",
			"gadget": gadgetName,
		})
		return
	}
	
	// Execute the gadget
    start := time.Now()
    execID := fmt.Sprintf("%s-%d", gadgetName, start.UnixNano())
    logging.L().Infow("gadget.exec.start",
        "request_id", c.GetString(logging.RequestIDKey),
        "exec_id", execID,
        "gadget_name", gadgetName,
        "args_count", len(req.Args),
        "user", claims.Username,
    )

    response := gi.executeGadgetCommand(gadgetName, req.Args, claims.Username)

    logging.L().Infow("gadget.exec.finish",
        "request_id", c.GetString(logging.RequestIDKey),
        "exec_id", execID,
        "gadget_name", gadgetName,
        "success", response.Success,
        "exit_code", response.ExitCode,
        "duration_ms", time.Since(start).Milliseconds(),
    )
	
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusInternalServerError, response)
	}
}

// ExecuteSystemGadget executes system-level gadgets (admin only)
func (gi *GadgetIntegration) ExecuteSystemGadget(c *gin.Context) {
	gadgetName := c.Param("name")
	
	if !gi.isSystemGadget(gadgetName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a system gadget"})
		return
	}
	
    var req GadgetExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	claims, _ := auth.GetUserFromContext(c)
	
    start := time.Now()
    execID := fmt.Sprintf("%s-%d", gadgetName, start.UnixNano())
    logging.L().Infow("gadget.exec.system.start",
        "request_id", c.GetString(logging.RequestIDKey),
        "exec_id", execID,
        "gadget_name", gadgetName,
        "args_count", len(req.Args),
        "user", claims.Username,
    )
    response := gi.executeGadgetCommand(gadgetName, req.Args, claims.Username)
    logging.L().Infow("gadget.exec.system.finish",
        "request_id", c.GetString(logging.RequestIDKey),
        "exec_id", execID,
        "gadget_name", gadgetName,
        "success", response.Success,
        "exit_code", response.ExitCode,
        "duration_ms", time.Since(start).Milliseconds(),
    )
	
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusInternalServerError, response)
	}
}

// executeGadgetCommand executes a gadget command with security measures
func (gi *GadgetIntegration) executeGadgetCommand(gadgetName string, args []string, username string) *GadgetExecuteResponse {
	// Prepare command
	cmdArgs := []string{"run", gadgetName}
	cmdArgs = append(cmdArgs, args...)
	
	// Create command with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30 second timeout
	defer cancel()
	
	cmd := exec.CommandContext(ctx, gi.gadgetBinaryPath, cmdArgs...)
	
	// Execute command
	output, err := cmd.CombinedOutput()
	
	response := &GadgetExecuteResponse{
		GadgetName: gadgetName,
		Output:     string(output),
	}
	
	if err != nil {
		response.Success = false
		response.Error = err.Error()
		
		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			response.ExitCode = exitError.ExitCode()
		} else {
			response.ExitCode = -1
		}
	} else {
		response.Success = true
		response.ExitCode = 0
	}
	
	// TODO: Add audit logging here
	// gi.auditLogger.LogGadgetExecution(username, gadgetName, args, response.Success)
	
	return response
}

// isValidGadgetName validates gadget names to prevent injection attacks
func (gi *GadgetIntegration) isValidGadgetName(name string) bool {
	// Allow only alphanumeric characters, hyphens, and underscores
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
			 (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return len(name) > 0 && len(name) <= 50
}

// isSystemGadget checks if a gadget is a system-level gadget
func (gi *GadgetIntegration) isSystemGadget(name string) bool {
	systemGadgets := []string{"sysinfo", "network-scanner", "process", "hardware"}
	for _, sg := range systemGadgets {
		if strings.EqualFold(name, sg) {
			return true
		}
	}
	return false
}

// parseGadgetList parses the output from "go-go-gadget list"
func (gi *GadgetIntegration) parseGadgetList(output string) []GadgetInfo {
	// This is a simplified parser - would need to be adapted based on actual output format
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var gadgets []GadgetInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Go Go Gadget") {
			continue
		}
		
		// Simple parsing - adjust based on actual format
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			gadget := GadgetInfo{
				Name:        parts[0],
				Description: strings.Join(parts[1:], " "),
			}
			gadgets = append(gadgets, gadget)
		}
	}
	
	return gadgets
}

// parseGadgetInfo parses the output from "go-go-gadget info <name>"
func (gi *GadgetIntegration) parseGadgetInfo(output, name string) *GadgetInfo {
	// Simplified parser - would need to be adapted based on actual output format
	return &GadgetInfo{
		Name:        name,
		Description: strings.TrimSpace(output),
		Category:    "general",
		Version:     "1.0.0",
	}
}

// GetGadgetsBridge creates a bridge function for MCP integration
func (gi *GadgetIntegration) GetGadgetsBridge() map[string]interface{} {
	return map[string]interface{}{
		"list_gadgets": func() ([]GadgetInfo, error) {
			cmd := exec.Command(gi.gadgetBinaryPath, "list")
			output, err := cmd.Output()
			if err != nil {
				return nil, err
			}
			return gi.parseGadgetList(string(output)), nil
		},
		
		"execute_gadget": func(name string, args []string) (*GadgetExecuteResponse, error) {
			if !gi.isValidGadgetName(name) {
				return nil, fmt.Errorf("invalid gadget name")
			}
			
			response := gi.executeGadgetCommand(name, args, "mcp-bridge")
			if !response.Success {
				return response, fmt.Errorf("gadget execution failed: %s", response.Error)
			}
			return response, nil
		},
	}
}

// HealthCheck provides a health check for the gadget integration
func (gi *GadgetIntegration) HealthCheck() error {
	cmd := exec.Command(gi.gadgetBinaryPath, "list")
	_, err := cmd.Output()
	return err
}