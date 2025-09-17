package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/chromedp/chromedp"
	"github.com/viper-org/viper/internal/types"
)

type Server struct {
	listen   string
	vmName   string
	taskDir  string
	startTime time.Time

	mu       sync.RWMutex
	contexts map[string]*BrowserContext
	tasks    map[string]*types.Task

	engine *gin.Engine
	server *http.Server
}

type BrowserContext struct {
	ID       string
	Context  context.Context
	Cancel   context.CancelFunc
	Profile  *types.Profile
	Created  time.Time
	LastUsed *time.Time
}

func NewServer(listen, vmName, taskDir string) (*Server, error) {
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create task directory: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		listen:    listen,
		vmName:    vmName,
		taskDir:   taskDir,
		startTime: time.Now(),
		contexts:  make(map[string]*BrowserContext),
		tasks:     make(map[string]*types.Task),
		engine:    gin.New(),
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	s.engine.Use(gin.Logger())
	s.engine.Use(gin.Recovery())

	s.engine.GET("/health", s.handleHealth)
	s.engine.POST("/spawn/:id", s.handleSpawnContext)
	s.engine.GET("/contexts", s.handleListContexts)
	s.engine.DELETE("/contexts/:id", s.handleDestroyContext)
	s.engine.POST("/profile/:ctxid", s.handleAttachProfile)
	s.engine.POST("/task", s.handleSubmitTask)
	s.engine.GET("/task/:id", s.handleGetTaskStatus)
	s.engine.GET("/logs/:taskid", s.handleGetTaskLogs)
	s.engine.GET("/screenshots/:taskid", s.handleGetTaskScreenshots)
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    s.listen,
		Handler: s.engine,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.mu.Lock()
	for _, ctx := range s.contexts {
		if ctx.Cancel != nil {
			ctx.Cancel()
		}
	}
	s.mu.Unlock()

	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	s.mu.RLock()
	contextCount := len(s.contexts)
	taskCount := len(s.tasks)
	s.mu.RUnlock()

	health := types.AgentHealth{
		Status:    "healthy",
		Version:   "0.1.0",
		Uptime:    time.Since(s.startTime),
		Contexts:  contextCount,
		Tasks:     taskCount,
		Memory:    int64(m.Alloc),
		LastCheck: time.Now(),
		Details: map[string]string{
			"vm_name": s.vmName,
		},
	}

	c.JSON(http.StatusOK, health)
}

func (s *Server) handleSpawnContext(c *gin.Context) {
	contextID := c.Param("id")

	s.mu.Lock()
	if _, exists := s.contexts[contextID]; exists {
		s.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "context already exists"})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())

	browserCtx := &BrowserContext{
		ID:      contextID,
		Context: ctx,
		Cancel:  cancel,
		Created: time.Now(),
	}

	s.contexts[contextID] = browserCtx
	s.mu.Unlock()

	log.Printf("Spawned browser context: %s", contextID)
	c.JSON(http.StatusOK, gin.H{"status": "spawned", "id": contextID})
}

func (s *Server) handleListContexts(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contexts := make([]types.BrowserContext, 0, len(s.contexts))
	for _, ctx := range s.contexts {
		contexts = append(contexts, types.BrowserContext{
			ID:       ctx.ID,
			VMID:     s.vmName,
			Created:  ctx.Created,
			Profile:  ctx.Profile,
			Active:   true,
			LastUsed: ctx.LastUsed,
		})
	}

	c.JSON(http.StatusOK, contexts)
}

func (s *Server) handleDestroyContext(c *gin.Context) {
	contextID := c.Param("id")

	s.mu.Lock()
	ctx, exists := s.contexts[contextID]
	if !exists {
		s.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "context not found"})
		return
	}

	if ctx.Cancel != nil {
		ctx.Cancel()
	}

	delete(s.contexts, contextID)
	s.mu.Unlock()

	log.Printf("Destroyed browser context: %s", contextID)
	c.JSON(http.StatusOK, gin.H{"status": "destroyed", "id": contextID})
}

func (s *Server) handleAttachProfile(c *gin.Context) {
	contextID := c.Param("ctxid")

	var profile types.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	browserCtx, exists := s.contexts[contextID]
	if !exists {
		s.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "context not found"})
		return
	}

	browserCtx.Profile = &profile
	s.mu.Unlock()

	if err := s.applyProfile(browserCtx.Context, profile); err != nil {
		log.Printf("Failed to apply profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Attached profile %s to context %s", profile.Name, contextID)
	c.JSON(http.StatusOK, gin.H{"status": "profile attached", "id": profile.ID})
}

func (s *Server) applyProfile(ctx context.Context, profile types.Profile) error {
	var tasks []chromedp.Action

	if profile.Viewport != nil {
		tasks = append(tasks, chromedp.EmulateViewport(profile.Viewport.Width, profile.Viewport.Height))
	}

	if profile.UserAgent != "" {
		script := fmt.Sprintf(`Object.defineProperty(navigator, 'userAgent', {get: function() { return '%s'; }});`, profile.UserAgent)
		tasks = append(tasks, chromedp.Evaluate(script, nil))
	}

	if len(tasks) > 0 {
		if err := chromedp.Run(ctx, tasks...); err != nil {
			return fmt.Errorf("failed to apply profile settings: %w", err)
		}
	}

	for domain, kv := range profile.LocalStorage {
		for key, value := range kv {
			script := fmt.Sprintf(`localStorage.setItem("%s", "%s")`, key, value)
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
				log.Printf("Failed to set localStorage %s=%s for %s: %v", key, value, domain, err)
			}
		}
	}

	return nil
}

func (s *Server) handleSubmitTask(c *gin.Context) {
	var task types.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	task.Status = types.TaskStatusPending
	task.Created = time.Now()

	s.mu.Lock()
	s.tasks[task.ID] = &task
	s.mu.Unlock()

	go s.executeTask(&task)

	result := types.TaskResult{
		TaskID: task.ID,
		Status: task.Status,
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) executeTask(task *types.Task) {
	now := time.Now()
	task.Started = &now
	task.Status = types.TaskStatusRunning

	defer func() {
		completed := time.Now()
		task.Completed = &completed
		if task.Status == types.TaskStatusRunning {
			task.Status = types.TaskStatusCompleted
		}

		s.mu.Lock()
		s.tasks[task.ID] = task
		s.mu.Unlock()
	}()

	taskDir := filepath.Join(s.taskDir, task.VMID, task.ID)
	if err := os.MkdirAll(filepath.Join(taskDir, "screenshots"), 0755); err != nil {
		task.Error = fmt.Sprintf("failed to create task directory: %v", err)
		task.Status = types.TaskStatusFailed
		return
	}

	logFile, err := os.Create(filepath.Join(taskDir, "stdout.log"))
	if err != nil {
		task.Error = fmt.Sprintf("failed to create log file: %v", err)
		task.Status = types.TaskStatusFailed
		return
	}
	defer logFile.Close()

	s.mu.RLock()
	browserCtx, exists := s.contexts[task.VMID]
	s.mu.RUnlock()

	if !exists {
		task.Error = "browser context not found"
		task.Status = types.TaskStatusFailed
		return
	}

	now = time.Now()
	browserCtx.LastUsed = &now

	timeout := 60 * time.Second
	if task.Timeout > 0 {
		timeout = task.Timeout
	}

	ctx, cancel := context.WithTimeout(browserCtx.Context, timeout)
	defer cancel()

	var screenshot []byte

	actions := []chromedp.Action{
		chromedp.Navigate(task.URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.CaptureScreenshot(&screenshot),
	}

	if task.Script != "" {
		actions = append(actions, chromedp.Evaluate(task.Script, nil))
	}

	err = chromedp.Run(ctx, actions...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			task.Error = "task timeout"
			task.Status = types.TaskStatusTimeout
		} else {
			task.Error = err.Error()
			task.Status = types.TaskStatusFailed
		}
		return
	}

	screenshotPath := filepath.Join(taskDir, "screenshots", "1.png")
	if err := os.WriteFile(screenshotPath, screenshot, 0644); err != nil {
		log.Printf("Failed to save screenshot: %v", err)
	}

	metadata := map[string]string{
		"url":    task.URL,
		"status": "completed",
	}
	metaData, _ := json.Marshal(metadata)
	os.WriteFile(filepath.Join(taskDir, "metadata.json"), metaData, 0644)

	logFile.WriteString(fmt.Sprintf("Task %s completed successfully\n", task.ID))
	logFile.WriteString(fmt.Sprintf("URL: %s\n", task.URL))
	logFile.WriteString(fmt.Sprintf("Screenshot saved: %s\n", screenshotPath))

	log.Printf("Task %s completed successfully", task.ID)
}

func (s *Server) handleGetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) handleGetTaskLogs(c *gin.Context) {
	taskID := c.Param("taskid")
	logPath := filepath.Join(s.taskDir, s.vmName, taskID, "stdout.log")

	data, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "logs not found"})
		return
	}

	c.Data(http.StatusOK, "text/plain", data)
}

func (s *Server) handleGetTaskScreenshots(c *gin.Context) {
	taskID := c.Param("taskid")
	screenshotsPath := filepath.Join(s.taskDir, s.vmName, taskID, "screenshots")

	entries, err := os.ReadDir(screenshotsPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "screenshots not found"})
		return
	}

	var screenshots []string
	for _, entry := range entries {
		if !entry.IsDir() {
			screenshots = append(screenshots, entry.Name())
		}
	}

	c.JSON(http.StatusOK, screenshots)
}