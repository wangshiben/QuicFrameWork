package server

import (
	"context"
	"time"
)

// ScheduledTask starts session maintenance once. Close and Shutdown stop it;
// framework code does not subscribe to process-wide operating-system signals.
func (s *Server) ScheduledTask() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.taskCtx == nil {
		s.taskCtx, s.taskCancel = context.WithCancel(context.Background())
	}
	if s.taskCtx.Err() != nil || s.Session == nil {
		return
	}
	s.taskOnce.Do(func() {
		session := s.Session
		interval := session.GetNextTimePicker()
		if interval <= 0 {
			interval = time.Minute
		}
		s.taskWG.Add(1)
		go func() {
			defer s.taskWG.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-s.taskCtx.Done():
					return
				case <-ticker.C:
					session.CleanExpItem()
				}
			}
		}()
	})
}
