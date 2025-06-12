package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"var-sync/internal/config"
	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/pkg/models"
)

// BenchmarkParserLoadFile benchmarks file loading performance
func BenchmarkParserLoadFile(b *testing.B) {
	tempDir := b.TempDir()
	
	testCases := []struct {
		name    string
		format  string
		content string
	}{
		{
			name:   "JSON",
			format: "json",
			content: `{
				"database": {
					"host": "localhost",
					"port": 5432,
					"connections": 100
				},
				"api": {
					"endpoints": [
						{"path": "/users", "method": "GET"},
						{"path": "/users", "method": "POST"},
						{"path": "/users/{id}", "method": "GET"},
						{"path": "/users/{id}", "method": "PUT"},
						{"path": "/users/{id}", "method": "DELETE"}
					]
				}
			}`,
		},
		{
			name:   "YAML",
			format: "yaml",
			content: `database:
  host: localhost
  port: 5432
  connections: 100
api:
  endpoints:
    - path: /users
      method: GET
    - path: /users
      method: POST
    - path: /users/{id}
      method: GET
    - path: /users/{id}
      method: PUT
    - path: /users/{id}
      method: DELETE`,
		},
		{
			name:   "TOML",
			format: "toml",
			content: `[database]
host = "localhost"
port = 5432
connections = 100

[[api.endpoints]]
path = "/users"
method = "GET"

[[api.endpoints]]
path = "/users"
method = "POST"

[[api.endpoints]]
path = "/users/{id}"
method = "GET"

[[api.endpoints]]
path = "/users/{id}"
method = "PUT"

[[api.endpoints]]
path = "/users/{id}"
method = "DELETE"`,
		},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("test.%s", tc.format))
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				b.Fatalf("Failed to create test file: %v", err)
			}
			
			parser := parser.New()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := parser.LoadFile(filePath)
				if err != nil {
					b.Fatalf("LoadFile failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkParserGetValue benchmarks value retrieval performance
func BenchmarkParserGetValue(b *testing.B) {
	data := createLargeTestData()
	parser := parser.New()
	
	testCases := []struct {
		name    string
		keyPath string
	}{
		{"ShallowKey", "simple_key"},
		{"DeepKey", "level1.level2.level3.level4.deep_value"},
		{"ArrayKey", "api.base_url"},
		{"MidLevelKey", "database.config.timeout"},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := parser.GetValue(data, tc.keyPath)
				if err != nil {
					b.Fatalf("GetValue failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkParserSetValue benchmarks value setting performance
func BenchmarkParserSetValue(b *testing.B) {
	parser := parser.New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := createLargeTestData()
		err := parser.SetValue(data, "new.nested.key", "new_value")
		if err != nil {
			b.Fatalf("SetValue failed: %v", err)
		}
	}
}

// BenchmarkParserSaveFile benchmarks file saving performance
func BenchmarkParserSaveFile(b *testing.B) {
	tempDir := b.TempDir()
	data := createLargeTestData()
	parser := parser.New()
	
	formats := []string{"json", "yaml", "toml"}
	
	for _, format := range formats {
		b.Run(format, func(b *testing.B) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("bench.%s", format))
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := parser.SaveFile(filePath, data)
				if err != nil {
					b.Fatalf("SaveFile failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConfigOperations benchmarks config management operations
func BenchmarkConfigOperations(b *testing.B) {
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "bench-config.json")
	
	b.Run("CreateManager", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manager, err := config.NewManager(configFile)
			if err != nil {
				b.Fatalf("NewManager failed: %v", err)
			}
			_ = manager
		}
	})
	
	b.Run("AddRule", func(b *testing.B) {
		manager, err := config.NewManager(configFile)
		if err != nil {
			b.Fatalf("NewManager failed: %v", err)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rule := models.SyncRule{
				ID:         fmt.Sprintf("rule-%d", i),
				Name:       fmt.Sprintf("Benchmark Rule %d", i),
				SourceFile: "source.yaml",
				SourceKey:  "key",
				TargetFile: "target.json",
				TargetKey:  "key",
				Enabled:    true,
				Created:    time.Now(),
			}
			manager.AddRule(rule)
		}
	})
	
	b.Run("GetRule", func(b *testing.B) {
		manager, err := config.NewManager(configFile)
		if err != nil {
			b.Fatalf("NewManager failed: %v", err)
		}
		
		// Add some rules first
		for i := 0; i < 100; i++ {
			rule := models.SyncRule{
				ID:         fmt.Sprintf("rule-%d", i),
				Name:       fmt.Sprintf("Rule %d", i),
				SourceFile: "source.yaml",
				SourceKey:  "key",
				TargetFile: "target.json",
				TargetKey:  "key",
				Enabled:    true,
				Created:    time.Now(),
			}
			manager.AddRule(rule)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ruleID := fmt.Sprintf("rule-%d", i%100)
			rule := manager.GetRule(ruleID)
			if rule == nil {
				b.Fatalf("GetRule failed for %s", ruleID)
			}
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent access patterns
func BenchmarkConcurrentOperations(b *testing.B) {
	tempDir := b.TempDir()
	
	b.Run("ConcurrentFileReads", func(b *testing.B) {
		// Create test files
		files := make([]string, 10)
		for i := 0; i < 10; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d.json", i))
			content := fmt.Sprintf(`{"id": %d, "data": "test-data-%d"}`, i, i)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				b.Fatalf("Failed to create test file: %v", err)
			}
			files[i] = filePath
		}
		
		parser := parser.New()
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, file := range files {
					_, err := parser.LoadFile(file)
					if err != nil {
						b.Fatalf("LoadFile failed: %v", err)
					}
				}
			}
		})
	})
	
	b.Run("ConcurrentDataManipulation", func(b *testing.B) {
		parser := parser.New()
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			// Each goroutine gets its own data to avoid concurrent map access
			data := createLargeTestData()
			i := 0
			for pb.Next() {
				// Alternate between reading and writing on separate data
				if i%2 == 0 {
					_, err := parser.GetValue(data, "database.host")
					if err != nil {
						b.Fatalf("GetValue failed: %v", err)
					}
				} else {
					err := parser.SetValue(data, fmt.Sprintf("temp_key_%d", i), fmt.Sprintf("temp_value_%d", i))
					if err != nil {
						b.Fatalf("SetValue failed: %v", err)
					}
				}
				i++
			}
		})
	})
}

// TestPerformanceMemoryUsage tests memory usage under load
func TestPerformanceMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}
	
	tempDir := t.TempDir()
	
	// Measure memory before test
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	// Create and process many files
	parser := parser.New()
	logger := logger.New()
	
	const numFiles = 1000
	const numOperations = 10000
	
	// Create test files
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("perf-test-%d.json", i))
		content := fmt.Sprintf(`{
			"id": %d,
			"timestamp": "%s",
			"data": {
				"nested": {
					"value": "test-value-%d",
					"count": %d
				}
			}
		}`, i, time.Now().Format(time.RFC3339), i, i*10)
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
		files[i] = filePath
	}
	
	// Perform operations
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		fileIndex := i % numFiles
		
		// Load file
		data, err := parser.LoadFile(files[fileIndex])
		if err != nil {
			t.Fatalf("LoadFile failed on iteration %d: %v", i, err)
		}
		
		// Get value
		_, err = parser.GetValue(data, "data.nested.value")
		if err != nil {
			t.Fatalf("GetValue failed on iteration %d: %v", i, err)
		}
		
		// Set value
		err = parser.SetValue(data, "data.nested.modified", time.Now().Unix())
		if err != nil {
			t.Fatalf("SetValue failed on iteration %d: %v", i, err)
		}
		
		// Save file (every 10th iteration to reduce I/O)
		if i%10 == 0 {
			err = parser.SaveFile(files[fileIndex], data)
			if err != nil {
				t.Fatalf("SaveFile failed on iteration %d: %v", i, err)
			}
		}
		
		// Log progress (every 1000 operations)
		if i%1000 == 0 {
			logger.Debug("Completed %d operations", i)
		}
	}
	
	duration := time.Since(start)
	
	// Measure memory after test
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	
	// Calculate metrics
	opsPerSecond := float64(numOperations) / duration.Seconds()
	memUsed := memAfter.Alloc - memBefore.Alloc
	
	logger.Info("Performance test completed:")
	logger.Info("  Operations: %d", numOperations)
	logger.Info("  Duration: %v", duration)
	logger.Info("  Ops/sec: %.2f", opsPerSecond)
	logger.Info("  Memory used: %d bytes (%.2f MB)", memUsed, float64(memUsed)/1024/1024)
	
	// Performance assertions (adjust thresholds as needed)
	if opsPerSecond < 100 {
		t.Errorf("Performance too slow: %.2f ops/sec (expected > 100)", opsPerSecond)
	}
	
	maxMemoryMB := float64(100) // 100MB threshold
	actualMemoryMB := float64(memUsed) / 1024 / 1024
	if actualMemoryMB > maxMemoryMB {
		t.Errorf("Memory usage too high: %.2f MB (expected < %.2f MB)", actualMemoryMB, maxMemoryMB)
	}
}

// TestPerformanceConcurrency tests performance under concurrent load
func TestPerformanceConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}
	
	tempDir := t.TempDir()
	
	// Create shared test files
	numFiles := 50
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d.json", i))
		content := fmt.Sprintf(`{"worker_id": %d, "counter": 0, "data": "initial"}`, i)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
		files[i] = filePath
	}
	
	// Test concurrent workers
	numWorkers := 20
	operationsPerWorker := 500
	
	var wg sync.WaitGroup
	start := time.Now()
	
	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			parser := parser.New()
			log := logger.New()
			log.SetLevel(logger.ERROR) // Reduce log noise
			
			for i := 0; i < operationsPerWorker; i++ {
				fileIndex := (id*operationsPerWorker + i) % numFiles
				
				// Load file
				data, err := parser.LoadFile(files[fileIndex])
				if err != nil {
					t.Errorf("Worker %d: LoadFile failed: %v", id, err)
					return
				}
				
				// Update counter
				counter, _ := parser.GetValue(data, "counter")
				if counter == nil {
					counter = 0
				}
				
				newCounter := 1
				if c, ok := counter.(float64); ok {
					newCounter = int(c) + 1
				}
				
				err = parser.SetValue(data, "counter", newCounter)
				if err != nil {
					t.Errorf("Worker %d: SetValue failed: %v", id, err)
					return
				}
				
				err = parser.SetValue(data, "last_worker", id)
				if err != nil {
					t.Errorf("Worker %d: SetValue failed: %v", id, err)
					return
				}
				
				// Save file (every 10th operation)
				if i%10 == 0 {
					err = parser.SaveFile(files[fileIndex], data)
					if err != nil {
						t.Errorf("Worker %d: SaveFile failed: %v", id, err)
						return
					}
				}
			}
		}(workerID)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	totalOperations := numWorkers * operationsPerWorker
	opsPerSecond := float64(totalOperations) / duration.Seconds()
	
	t.Logf("Concurrency test completed:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Operations per worker: %d", operationsPerWorker)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Ops/sec: %.2f", opsPerSecond)
	
	// Verify no data corruption
	parser := parser.New()
	for i, file := range files {
		data, err := parser.LoadFile(file)
		if err != nil {
			t.Errorf("Failed to verify file %d: %v", i, err)
			continue
		}
		
		counter, err := parser.GetValue(data, "counter")
		if err != nil {
			t.Errorf("Failed to get counter from file %d: %v", i, err)
		}
		
		// Counter should be positive (multiple workers incremented it)
		if c, ok := counter.(float64); ok && c <= 0 {
			t.Errorf("File %d has invalid counter: %v", i, c)
		}
	}
}

// Helper function to create large test data
func createLargeTestData() map[string]any {
	data := map[string]any{
		"simple_key": "simple_value",
		"database": map[string]any{
			"host":     "localhost",
			"port":     5432,
			"username": "admin",
			"config": map[string]any{
				"timeout":     30,
				"connections": 100,
				"ssl":         true,
			},
		},
		"api": map[string]any{
			"base_url": "https://api.example.com",
			"version":  "v2",
			"endpoints": []any{
				map[string]any{"path": "/users", "method": "GET"},
				map[string]any{"path": "/users", "method": "POST"},
				map[string]any{"path": "/posts", "method": "GET"},
			},
		},
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com"},
			map[string]any{"id": 2, "name": "Bob", "email": "bob@example.com"},
			map[string]any{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		},
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"level4": map[string]any{
						"deep_value": "found_it",
					},
				},
			},
		},
	}
	
	// Add more data to make it larger
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("dynamic_key_%d", i)
		data[key] = map[string]any{
			"id":    i,
			"value": fmt.Sprintf("dynamic_value_%d", i),
			"metadata": map[string]any{
				"created": time.Now().Format(time.RFC3339),
				"type":    "dynamic",
			},
		}
	}
	
	return data
}