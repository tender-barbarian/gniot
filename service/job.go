package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func (s *Service) RunJobs(ctx context.Context, interval time.Duration, errCh chan<- error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processJobs(ctx); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}
	}
}

func (s *Service) processJobs(ctx context.Context) error {
	jobs, err := s.jobsRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("getting jobs: %w", err)
	}

	now := time.Now()

	for _, job := range jobs {
		jobTime, err := time.Parse(time.RFC3339, job.RunAt)
		if err != nil {
			s.logger.Error("parsing job time", "job_id", job.ID, "error", err)
			continue
		}

		if jobTime.After(now) {
			continue
		}

		var deviceIds []int
		if err := json.Unmarshal([]byte(job.Devices), &deviceIds); err != nil {
			s.logger.Error("parsing job devices", "job_id", job.ID, "error", err)
			continue
		}

		actionId, err := strconv.Atoi(job.Action)
		if err != nil {
			s.logger.Error("parsing job action", "job_id", job.ID, "error", err)
			continue
		}

		for _, deviceId := range deviceIds {
			err := s.Execute(ctx, deviceId, actionId)
			if err != nil {
				s.logger.Error("executing job", "job_id", job.ID, "device_id", deviceId, "error", err)
			} else {
				s.logger.Info("succesfully executed job", "job_name", job.Name, "deviceId", deviceId)
			}
		}

		interval, err := time.ParseDuration(job.Interval)
		if err != nil {
			s.logger.Error("parsing job interval", "job_id", job.ID, "error", err)
			continue
		}

		job.RunAt = now.Add(interval).Format(time.RFC3339)
		err = s.jobsRepo.Update(ctx, job, job.ID)
		if err != nil {
			s.logger.Error("updating job time", "job_id", job.ID, "error", err)
		} else {
			s.logger.Info("succesfully updated job time", "job_name", job.Name)
		}
	}

	return nil
}
