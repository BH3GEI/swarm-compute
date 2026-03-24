package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu sync.RWMutex

	workers   []Worker
	taskTypes map[string]TaskType
	jobs      map[string]*Job

	dataDir string
}

type storeSnapshot struct {
	Workers   []Worker            `json:"workers,omitempty"`
	TaskTypes map[string]TaskType `json:"taskTypes,omitempty"`
	Jobs      map[string]*Job     `json:"jobs,omitempty"`
}

func NewStore() *Store {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "/data"
	}
	s := &Store{
		taskTypes: map[string]TaskType{},
		jobs:      map[string]*Job{},
		dataDir:   dir,
	}
	if err := s.LoadFromDisk(); err != nil {
		logWarn("load store from disk failed", withErr(err))
	}
	return s
}

func (s *Store) dataPath() string {
	p := os.Getenv("STORE_PATH")
	if p != "" {
		return p
	}
	return filepath.Join(s.dataDir, "store.json")
}

func (s *Store) LoadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(s.dataPath()), 0755)

	b, err := os.ReadFile(s.dataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var snap storeSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}

	if snap.TaskTypes == nil {
		snap.TaskTypes = map[string]TaskType{}
	}
	if snap.Jobs == nil {
		snap.Jobs = map[string]*Job{}
	}

	s.workers = snap.Workers
	s.taskTypes = snap.TaskTypes
	s.jobs = snap.Jobs
	return nil
}

// saveLocked persists state atomically. Caller MUST hold write lock.
func (s *Store) saveLocked() error {
	path := s.dataPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	snap := storeSnapshot{
		Workers:   s.workers,
		TaskTypes: s.taskTypes,
		Jobs:      s.jobs,
	}

	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SaveToDisk takes a read lock, snapshots, then writes atomically.
func (s *Store) SaveToDisk() error {
	s.mu.RLock()
	snap := storeSnapshot{
		Workers:   s.workers,
		TaskTypes: s.taskTypes,
		Jobs:      s.jobs,
	}
	s.mu.RUnlock()

	path := s.dataPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) saveToDiskPublic() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

// ====== stats ======

type SystemStats struct {
	TotalJobs     int            `json:"totalJobs"`
	JobsByStatus  map[string]int `json:"jobsByStatus"`
	AvgDurationMs float64        `json:"avgDurationMs"`
	Workers       []WorkerStat   `json:"workers"`
}

type WorkerStat struct {
	ID        string  `json:"id"`
	TasksDone int     `json:"tasksDone"`
	AvgMs     float64 `json:"avgMs"`
}

func (s *Store) ComputeStats() SystemStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := SystemStats{
		TotalJobs:    len(s.jobs),
		JobsByStatus: map[string]int{},
	}

	var totalDur int64
	var doneCount int
	workerTasks := map[string][]int64{}

	for _, j := range s.jobs {
		stats.JobsByStatus[j.Status]++
		if j.Status == "done" && j.FinishedAt > 0 && j.CreatedAt > 0 {
			totalDur += j.FinishedAt - j.CreatedAt
			doneCount++
		}
		for _, t := range j.Tasks {
			if t.Status == "done" && t.WorkerID != "" {
				workerTasks[t.WorkerID] = append(workerTasks[t.WorkerID], t.DurationMs)
			}
		}
	}

	if doneCount > 0 {
		stats.AvgDurationMs = float64(totalDur) / float64(doneCount)
	}

	for wid, durations := range workerTasks {
		var sum int64
		for _, d := range durations {
			sum += d
		}
		avg := float64(0)
		if len(durations) > 0 {
			avg = float64(sum) / float64(len(durations))
		}
		stats.Workers = append(stats.Workers, WorkerStat{
			ID:        wid,
			TasksDone: len(durations),
			AvgMs:     avg,
		})
	}

	return stats
}
