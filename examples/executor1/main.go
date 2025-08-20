package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	ExecutionID string                 `json:"execution_id"`
	TaskID      string                 `json:"task_id"`
	TaskName    string                 `json:"task_name"`
	Parameters  map[string]interface{} `json:"parameters"`
	CallbackURL string                 `json:"callback_url"`
}

// CallbackRequest 回调请求
type CallbackRequest struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result"`
	Logs        string                 `json:"logs"`
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/execute", executeHandler)
	executorID := "executor-1754728012191-bqujf"
	_ = executorID
	executorName := "00001"
	_ = executorName

	port := ":9091"
	log.Printf("Sample executor listening on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received task: %s (ID: %s)", req.TaskName, req.ExecutionID)

	// 异步执行任务
	go func() {
		// 模拟任务执行时间
		duration := time.Duration(rand.Intn(10)+1) * time.Second
		time.Sleep(duration)

		// 随机决定成功或失败
		status := "success"
		logs := fmt.Sprintf("Task %s executed successfully in %v", req.TaskName, duration)

		if rand.Float32() < 0.1 { // 10%的失败率
			status = "failed"
			logs = fmt.Sprintf("Task %s failed after %v", req.TaskName, duration)
		}

		// 回调调度器
		callback := CallbackRequest{
			ExecutionID: req.ExecutionID,
			Status:      status,
			Result: map[string]interface{}{
				"duration":   duration.Seconds(),
				"parameters": req.Parameters,
			},
			Logs: logs,
		}

		if err := sendCallback(req.CallbackURL, callback); err != nil {
			log.Printf("Failed to send callback: %v", err)
		} else {
			log.Printf("Callback sent for execution %s", req.ExecutionID)
		}
	}()

	// 立即返回接受响应
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message":      "Task accepted",
		"execution_id": req.ExecutionID,
	})
}

func sendCallback(url string, callback CallbackRequest) error {
	jsonData, err := json.Marshal(callback)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return nil
}
