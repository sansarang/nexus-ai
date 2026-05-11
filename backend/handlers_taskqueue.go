//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Task Queue — Agent 장시간 작업 관리
//  우선순위: urgent > normal > background
//  실행 중 SSE로 진행상황 실시간 push
// ══════════════════════════════════════════════════════════════════

type TaskPriority int

const (
	PriorityBackground TaskPriority = 0
	PriorityNormal     TaskPriority = 1
	PriorityUrgent     TaskPriority = 2
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskDone      TaskStatus = "done"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

type AgentTask struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Priority    TaskPriority           `json:"priority"`
	Status      TaskStatus             `json:"status"`
	Progress    int                    `json:"progress"` // 0~100
	Message     string                 `json:"message"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	FinishedAt  *time.Time             `json:"finished_at,omitempty"`
	Result      map[string]any         `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Params      map[string]any         `json:"params,omitempty"`
	cancelCh    chan struct{}
	handlerFunc func(task *AgentTask)
}

func (t *AgentTask) Cancel() {
	select {
	case t.cancelCh <- struct{}{}:
	default:
	}
}

func (t *AgentTask) IsCancelled() bool {
	select {
	case <-t.cancelCh:
		return true
	default:
		return false
	}
}

// UpdateProgress: 진행상황 업데이트 + SSE push
func (t *AgentTask) UpdateProgress(pct int, msg string) {
	t.Progress = pct
	t.Message = msg
	globalTaskQueue.publishTaskEvent(t)
}

// ── 글로벌 태스크 큐 ────────────────────────────────────────────

type TaskQueue struct {
	mu       sync.RWMutex
	tasks    map[string]*AgentTask
	queue    chan *AgentTask
	subs     map[string]chan *AgentTask
	subMu    sync.Mutex
	maxConc  int // 최대 동시 실행 수
}

var globalTaskQueue = &TaskQueue{
	tasks:   make(map[string]*AgentTask),
	queue:   make(chan *AgentTask, 100),
	subs:    make(map[string]chan *AgentTask),
	maxConc: 3,
}

func initTaskQueue() {
	for i := 0; i < globalTaskQueue.maxConc; i++ {
		go globalTaskQueue.worker()
	}
}

func (q *TaskQueue) worker() {
	for task := range q.queue {
		now := time.Now()
		task.StartedAt = &now
		task.Status = TaskRunning
		q.publishTaskEvent(task)

		func() {
			defer func() {
				if r := recover(); r != nil {
					task.Status = TaskFailed
					task.Error = fmt.Sprintf("패닉: %v", r)
					fin := time.Now()
					task.FinishedAt = &fin
					q.publishTaskEvent(task)
				}
			}()
			task.handlerFunc(task)
		}()

		if task.Status == TaskRunning {
			fin := time.Now()
			task.FinishedAt = &fin
			task.Status = TaskDone
			task.Progress = 100
			q.publishTaskEvent(task)
		}
	}
}

func (q *TaskQueue) Enqueue(name string, priority TaskPriority, params map[string]any, fn func(*AgentTask)) *AgentTask {
	task := &AgentTask{
		ID:          fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Name:        name,
		Priority:    priority,
		Status:      TaskPending,
		CreatedAt:   time.Now(),
		Params:      params,
		cancelCh:    make(chan struct{}, 1),
		handlerFunc: fn,
	}

	q.mu.Lock()
	q.tasks[task.ID] = task
	q.mu.Unlock()

	q.publishTaskEvent(task)
	q.queue <- task
	return task
}

func (q *TaskQueue) GetTask(id string) (*AgentTask, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	t, ok := q.tasks[id]
	return t, ok
}

func (q *TaskQueue) publishTaskEvent(task *AgentTask) {
	q.subMu.Lock()
	for _, ch := range q.subs {
		select {
		case ch <- task:
		default:
		}
	}
	q.subMu.Unlock()

	// Proactive 알림으로도 push (완료/실패 시)
	if task.Status == TaskDone || task.Status == TaskFailed {
		level := "info"
		if task.Status == TaskFailed {
			level = "warn"
		}
		msg := fmt.Sprintf("작업 완료: %s", task.Name)
		if task.Status == TaskFailed {
			msg = fmt.Sprintf("작업 실패: %s — %s", task.Name, task.Error)
		}
		publishAlert(Alert{
			ID:      "task_" + task.ID,
			Level:   level,
			Title:   task.Name,
			Message: msg,
		})
	}
}

// ── Task Queue HTTP 핸들러 ──────────────────────────────────────

// GET /api/tasks/stream — SSE로 태스크 진행상황 실시간 수신
func handleTaskStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan *AgentTask, 20)
	id := fmt.Sprintf("tsub_%d", time.Now().UnixNano())

	globalTaskQueue.subMu.Lock()
	globalTaskQueue.subs[id] = ch
	globalTaskQueue.subMu.Unlock()

	defer func() {
		globalTaskQueue.subMu.Lock()
		delete(globalTaskQueue.subs, id)
		globalTaskQueue.subMu.Unlock()
		close(ch)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(map[string]any{
				"type":        "task_update",
				"id":          task.ID,
				"name":        task.Name,
				"status":      task.Status,
				"progress":    task.Progress,
				"message":     task.Message,
				"error":       task.Error,
				"result":      task.Result,
				"finished_at": task.FinishedAt,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-time.After(25 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// GET /api/tasks/list
func handleTaskList(w http.ResponseWriter, r *http.Request) {
	globalTaskQueue.mu.RLock()
	tasks := make([]*AgentTask, 0, len(globalTaskQueue.tasks))
	for _, t := range globalTaskQueue.tasks {
		tasks = append(tasks, t)
	}
	globalTaskQueue.mu.RUnlock()

	json200(w, map[string]any{"tasks": tasks, "count": len(tasks)})
}

// POST /api/tasks/cancel — body: {id: "task_xxx"}
func handleTaskCancel(w http.ResponseWriter, r *http.Request) {
	var req struct{ ID string `json:"id"` }
	json.NewDecoder(r.Body).Decode(&req)
	task, ok := globalTaskQueue.GetTask(req.ID)
	if !ok {
		json200(w, map[string]any{"success": false, "message": "태스크를 찾을 수 없어요"})
		return
	}
	task.Cancel()
	task.Status = TaskCancelled
	fin := time.Now()
	task.FinishedAt = &fin
	json200(w, map[string]any{"success": true, "message": "태스크 취소 완료"})
}
