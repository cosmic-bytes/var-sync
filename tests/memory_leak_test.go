// +build memory

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"var-sync/internal/config"
	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/pkg/models"
)

// TestMemoryLeakParser tests for memory leaks in parser operations
func TestMemoryLeakParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	
	tempDir := t.TempDir()
	
	// Create test files
	jsonFile := filepath.Join(tempDir, "test.json")
	yamlFile := filepath.Join(tempDir, "test.yaml")
	tomlFile := filepath.Join(tempDir, "test.toml")
	
	jsonContent := `{
		"database": {
			"host": "localhost",
			"port": 5432,
			"config": {
				"timeout": 30,
				"connections": 100
			}
		},
		"large_array": [` + generateLargeJSONArray(1000) + `]
	}`
	
	yamlContent := `database:
  host: localhost
  port: 5432
  config:
    timeout: 30
    connections: 100
large_list:` + generateLargeYAMLList(1000)
	
	tomlContent := `[database]
host = "localhost"
port = 5432

[database.config]
timeout = 30
connections = 100
` + generateLargeTOMLArray(1000)
	
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to create JSON file: %v", err)
	}
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create YAML file: %v", err)
	}
	if err := os.WriteFile(tomlFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to create TOML file: %v", err)
	}
	
	files := []string{jsonFile, yamlFile, tomlFile}
	
	// Measure initial memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	parser := parser.New()
	
	// Perform many operations that could leak memory
	const iterations = 500
	for i := 0; i < iterations; i++ {
		for _, file := range files {
			// Load file
			data, err := parser.LoadFile(file)
			if err != nil {
				t.Fatalf("LoadFile failed on iteration %d: %v", i, err)
			}
			
			// Perform operations
			_, err = parser.GetValue(data, "database.host")
			if err != nil {
				t.Fatalf("GetValue failed on iteration %d: %v", i, err)
			}
			
			// Set multiple values
			for j := 0; j < 10; j++ {
				err = parser.SetValue(data, fmt.Sprintf("temp_key_%d_%d", i, j), fmt.Sprintf("temp_value_%d_%d", i, j))
				if err != nil {
					t.Fatalf("SetValue failed on iteration %d_%d: %v", i, j, err)
				}
			}
			
			// Get all keys (expensive operation)
			_ = parser.GetAllKeys(data, "")
			
			// Save file occasionally to test file operations
			if i%50 == 0 {
				tempFile := filepath.Join(tempDir, fmt.Sprintf("temp_%d_%s", i, filepath.Base(file)))
				err = parser.SaveFile(tempFile, data)
				if err != nil {
					t.Fatalf("SaveFile failed on iteration %d: %v", i, err)
				}
			}
		}
		
		// Force garbage collection every 100 iterations
		if i%100 == 0 {
			runtime.GC()
		}
	}
	
	// Final garbage collection
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup
	
	// Measure final memory
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	// Calculate memory growth
	memGrowth := int64(memAfter.Alloc) - int64(memBefore.Alloc)
	memGrowthMB := float64(memGrowth) / 1024 / 1024
	
	t.Logf("Memory leak test results:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Memory before: %d bytes", memBefore.Alloc)
	t.Logf("  Memory after: %d bytes", memAfter.Alloc)
	t.Logf("  Memory growth: %d bytes (%.2f MB)", memGrowth, memGrowthMB)
	t.Logf("  Total allocations: %d", memAfter.TotalAlloc)
	t.Logf("  Mallocs: %d", memAfter.Mallocs)
	t.Logf("  Frees: %d", memAfter.Frees)
	
	// Check for excessive memory growth (adjust threshold as needed)
	maxGrowthMB := float64(50) // 50MB threshold
	if memGrowthMB > maxGrowthMB {
		t.Errorf("Potential memory leak detected: memory grew by %.2f MB (threshold: %.2f MB)", memGrowthMB, maxGrowthMB)
	}
	
	// Check allocation/free ratio
	allocFreeRatio := float64(memAfter.Mallocs) / float64(memAfter.Frees)
	maxAllocFreeRatio := 1.1 // Allow 10% more allocations than frees
	if allocFreeRatio > maxAllocFreeRatio {
		t.Errorf("Poor allocation/free ratio: %.2f (threshold: %.2f)", allocFreeRatio, maxAllocFreeRatio)
	}
}

// TestMemoryLeakConfig tests for memory leaks in config operations
func TestMemoryLeakConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	
	tempDir := t.TempDir()
	
	// Measure initial memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	const iterations = 200
	for i := 0; i < iterations; i++ {
		configFile := filepath.Join(tempDir, fmt.Sprintf("config_%d.json", i))
		
		// Create config manager
		manager, err := config.NewManager(configFile)
		if err != nil {
			t.Fatalf("NewManager failed on iteration %d: %v", i, err)
		}
		
		// Add many rules
		for j := 0; j < 20; j++ {
			rule := models.SyncRule{
				ID:          fmt.Sprintf("rule_%d_%d", i, j),
				Name:        fmt.Sprintf("Rule %d-%d", i, j),
				Description: fmt.Sprintf("Test rule %d-%d for memory leak testing", i, j),
				SourceFile:  fmt.Sprintf("source_%d_%d.yaml", i, j),
				SourceKey:   fmt.Sprintf("key_%d_%d", i, j),
				TargetFile:  fmt.Sprintf("target_%d_%d.json", i, j),
				TargetKey:   fmt.Sprintf("target_key_%d_%d", i, j),
				Enabled:     true,
				Created:     time.Now(),
			}
			manager.AddRule(rule)
		}
		
		// Save config
		if err := manager.Save(); err != nil {
			t.Fatalf("Save failed on iteration %d: %v", i, err)
		}
		
		// Get rules
		for j := 0; j < 20; j++ {
			ruleID := fmt.Sprintf("rule_%d_%d", i, j)
			rule := manager.GetRule(ruleID)
			if rule == nil {
				t.Fatalf("GetRule failed for %s", ruleID)
			}
		}
		
		// Remove some rules
		for j := 0; j < 10; j++ {
			ruleID := fmt.Sprintf("rule_%d_%d", i, j)
			manager.RemoveRule(ruleID)
		}
		
		// Save again
		if err := manager.Save(); err != nil {
			t.Fatalf("Second save failed on iteration %d: %v", i, err)
		}
		
		// Force garbage collection every 50 iterations
		if i%50 == 0 {
			runtime.GC()
		}
	}
	
	// Final garbage collection
	runtime.GC()
	runtime.GC()
	
	// Measure final memory
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	// Calculate memory growth
	memGrowth := int64(memAfter.Alloc) - int64(memBefore.Alloc)
	memGrowthMB := float64(memGrowth) / 1024 / 1024
	
	t.Logf("Config memory leak test results:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Memory before: %d bytes", memBefore.Alloc)
	t.Logf("  Memory after: %d bytes", memAfter.Alloc)
	t.Logf("  Memory growth: %d bytes (%.2f MB)", memGrowth, memGrowthMB)
	
	// Check for excessive memory growth
	maxGrowthMB := float64(20) // 20MB threshold for config operations
	if memGrowthMB > maxGrowthMB {
		t.Errorf("Potential memory leak in config operations: memory grew by %.2f MB (threshold: %.2f MB)", memGrowthMB, maxGrowthMB)
	}
}

// TestMemoryLeakLogger tests for memory leaks in logging operations
func TestMemoryLeakLogger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "memory_test.log")
	
	// Measure initial memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	log := logger.New()
	log.SetLevel(logger.DEBUG)
	
	if err := log.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile failed: %v", err)
	}
	defer log.Close()
	
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		// Log at different levels
		log.Debug("Debug message %d with data: %s", i, generateLogData(i))
		log.Info("Info message %d with timestamp: %s", i, time.Now().Format(time.RFC3339))
		log.Warn("Warning message %d", i)
		log.Error("Error message %d with details: %v", i, map[string]interface{}{
			"iteration": i,
			"timestamp": time.Now().Unix(),
			"data":      generateLogData(i),
		})
		
		// Change log file occasionally
		if i%2000 == 0 && i > 0 {
			newLogFile := filepath.Join(tempDir, fmt.Sprintf("memory_test_%d.log", i))
			if err := log.SetLogFile(newLogFile); err != nil {
				t.Fatalf("SetLogFile failed on iteration %d: %v", i, err)
			}
		}
		
		// Force garbage collection every 1000 iterations
		if i%1000 == 0 {
			runtime.GC()
		}
	}
	
	// Final garbage collection
	runtime.GC()
	runtime.GC()
	
	// Measure final memory
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	// Calculate memory growth
	memGrowth := int64(memAfter.Alloc) - int64(memBefore.Alloc)
	memGrowthMB := float64(memGrowth) / 1024 / 1024
	
	t.Logf("Logger memory leak test results:")
	t.Logf("  Log messages: %d", iterations*4) // 4 messages per iteration
	t.Logf("  Memory before: %d bytes", memBefore.Alloc)
	t.Logf("  Memory after: %d bytes", memAfter.Alloc)
	t.Logf("  Memory growth: %d bytes (%.2f MB)", memGrowth, memGrowthMB)
	
	// Check for excessive memory growth
	maxGrowthMB := float64(10) // 10MB threshold for logging operations
	if memGrowthMB > maxGrowthMB {
		t.Errorf("Potential memory leak in logging: memory grew by %.2f MB (threshold: %.2f MB)", memGrowthMB, maxGrowthMB)
	}
}

// TestMemoryLeakLongRunning simulates long-running application behavior
func TestMemoryLeakLongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running memory leak test in short mode")
	}
	
	tempDir := t.TempDir()
	
	// Create test files
	sourceFile := filepath.Join(tempDir, "source.yaml")
	targetFile := filepath.Join(tempDir, "target.json")
	configFile := filepath.Join(tempDir, "config.json")
	logFile := filepath.Join(tempDir, "app.log")
	
	// Create source file
	sourceContent := `database:
  host: localhost
  port: 5432
api:
  key: secret123`
	
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Create target file
	targetContent := `{"config": {"db": {"host": "old"}}}`
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	// Create sync rule
	rule := models.SyncRule{
		ID:         "long-running-rule",
		Name:       "Long Running Test Rule",
		SourceFile: sourceFile,
		SourceKey:  "database.host",
		TargetFile: targetFile,
		TargetKey:  "config.db.host",
		Enabled:    true,
		Created:    time.Now(),
	}
	
	// Create config
	cfg := &models.Config{
		Rules:   []models.SyncRule{rule},
		LogFile: logFile,
		Debug:   false,
	}
	
	if err := config.Save(cfg, configFile); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Measure initial memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// Initialize components
	parser := parser.New()
	log := logger.New()
	log.SetLevel(logger.INFO)
	
	if err := log.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile failed: %v", err)
	}
	defer log.Close()
	
	// Simulate long-running sync operations
	const duration = 30 * time.Second // Run for 30 seconds
	const syncInterval = 100 * time.Millisecond
	
	start := time.Now()
	syncCount := 0
	memSamples := []uint64{}
	
	for time.Since(start) < duration {
		// Perform sync operation
		sourceData, err := parser.LoadFile(sourceFile)
		if err != nil {
			t.Fatalf("LoadFile failed after %v: %v", time.Since(start), err)
		}
		
		sourceValue, err := parser.GetValue(sourceData, rule.SourceKey)
		if err != nil {
			t.Fatalf("GetValue failed after %v: %v", time.Since(start), err)
		}
		
		targetData, err := parser.LoadFile(targetFile)
		if err != nil {
			t.Fatalf("LoadFile target failed after %v: %v", time.Since(start), err)
		}
		
		if err := parser.SetValue(targetData, rule.TargetKey, sourceValue); err != nil {
			t.Fatalf("SetValue failed after %v: %v", time.Since(start), err)
		}
		
		if err := parser.SaveFile(targetFile, targetData); err != nil {
			t.Fatalf("SaveFile failed after %v: %v", time.Since(start), err)
		}
		
		syncCount++
		
		// Log operation
		log.Info("Sync operation %d completed", syncCount)
		
		// Sample memory every 100 syncs
		if syncCount%100 == 0 {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			memSamples = append(memSamples, memStats.Alloc)
		}
		
		// Force garbage collection every 500 syncs
		if syncCount%500 == 0 {
			runtime.GC()
		}
		
		time.Sleep(syncInterval)
	}
	
	// Final garbage collection
	runtime.GC()
	runtime.GC()
	
	// Measure final memory
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	// Calculate memory metrics
	memGrowth := int64(memAfter.Alloc) - int64(memBefore.Alloc)
	memGrowthMB := float64(memGrowth) / 1024 / 1024
	avgMemGrowthPerSync := float64(memGrowth) / float64(syncCount)
	
	t.Logf("Long-running memory leak test results:")
	t.Logf("  Duration: %v", time.Since(start))
	t.Logf("  Sync operations: %d", syncCount)
	t.Logf("  Memory before: %d bytes", memBefore.Alloc)
	t.Logf("  Memory after: %d bytes", memAfter.Alloc)
	t.Logf("  Memory growth: %d bytes (%.2f MB)", memGrowth, memGrowthMB)
	t.Logf("  Avg memory growth per sync: %.2f bytes", avgMemGrowthPerSync)
	
	// Analyze memory samples for trends
	if len(memSamples) > 1 {
		firstSample := memSamples[0]
		lastSample := memSamples[len(memSamples)-1]
		sampleGrowth := int64(lastSample) - int64(firstSample)
		sampleGrowthMB := float64(sampleGrowth) / 1024 / 1024
		
		t.Logf("  Memory sample trend: %.2f MB growth over %d samples", sampleGrowthMB, len(memSamples))
		
		// Check for consistent memory growth (potential leak)
		if sampleGrowthMB > 5.0 { // 5MB growth in samples
			t.Errorf("Potential memory leak detected in samples: %.2f MB growth", sampleGrowthMB)
		}
	}
	
	// Check overall memory growth
	maxGrowthMB := float64(15) // 15MB threshold for long-running test
	if memGrowthMB > maxGrowthMB {
		t.Errorf("Potential memory leak in long-running test: memory grew by %.2f MB (threshold: %.2f MB)", memGrowthMB, maxGrowthMB)
	}
	
	// Check per-operation memory growth
	maxGrowthPerSync := 100.0 // 100 bytes per sync operation
	if avgMemGrowthPerSync > maxGrowthPerSync {
		t.Errorf("Excessive memory growth per sync: %.2f bytes (threshold: %.2f bytes)", avgMemGrowthPerSync, maxGrowthPerSync)
	}
}

// Helper functions to generate test data

func generateLargeJSONArray(size int) string {
	items := make([]string, size)
	for i := 0; i < size; i++ {
		items[i] = fmt.Sprintf(`{"id": %d, "name": "item_%d", "value": "data_%d"}`, i, i, i)
	}
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ","
		}
		result += item
	}
	return result
}

func generateLargeYAMLList(size int) string {
	result := "\n"
	for i := 0; i < size; i++ {
		result += fmt.Sprintf("  - id: %d\n    name: item_%d\n    value: data_%d\n", i, i, i)
	}
	return result
}

func generateLargeTOMLArray(size int) string {
	result := ""
	for i := 0; i < size; i++ {
		result += fmt.Sprintf("[[large_array]]\nid = %d\nname = \"item_%d\"\nvalue = \"data_%d\"\n\n", i, i, i)
	}
	return result
}

func generateLogData(iteration int) string {
	return fmt.Sprintf("iteration_%d_data_with_timestamp_%d", iteration, time.Now().UnixNano())
}