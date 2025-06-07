package workerPool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

type Job struct {
	Text string
}

type Worker struct {
	ID   int
	done chan struct{}
}

type WorkersPool struct {
	workers     []*Worker
	jobsCh      chan Job
	wg          *sync.WaitGroup
	done        chan struct{}
	maxJobCount int
}

func New(jobsCount int) *WorkersPool {
	file, _ := os.OpenFile("legacy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	log.SetOutput(file)
	return &WorkersPool{
		workers:     []*Worker{},
		maxJobCount: jobsCount,
		jobsCh:      make(chan Job, jobsCount),
		wg:          &sync.WaitGroup{},
		done:        make(chan struct{}),
	}
}

func (wp *WorkersPool) work(ctx context.Context, w *Worker) {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.done:
			log.Println("[DEBUG] Worker number " + strconv.Itoa(w.ID) + " was removed")
			return
		case <-w.done:
			log.Println("[DEBUG] Worker number " + strconv.Itoa(w.ID) + " was removed")
			return
		case <-ctx.Done():
			log.Printf("[DEBUG] Ctx was done: %s", ctx.Err().Error())
			return
		case job, ok := <-wp.jobsCh:
			if ok {
				fmt.Println("Worker " + strconv.Itoa(w.ID) + " done the job: " + job.Text)
				log.Println("[DEBUG] Worker number " + strconv.Itoa(w.ID) + " get done job: " + job.Text)
			} else {
				log.Println("[DEBUG] Worker number " + strconv.Itoa(w.ID) + " don't recieve job")
			}
		}
	}
}

func (wp *WorkersPool) AddJob(data string) error {
	if len(wp.jobsCh) == wp.maxJobCount {
		log.Println("[DEBUG] Job: " + data + " wasn't included in channel")
		return errors.New("Channel is full")
	}
	wp.jobsCh <- Job{Text: data}
	log.Println("[DEBUG] Job: " + data + " was included in channel")
	return nil
}

func (wp *WorkersPool) AddWorkers(ctx context.Context, count int) error {
	for i := 0; i < count; i++ {
		id := len(wp.workers)
		w := &Worker{ID: id, done: make(chan struct{})}
		wp.workers = append(wp.workers, w)
		wp.wg.Add(1)
		go wp.work(ctx, w)
		log.Println("[DEBUG] Worker number " + strconv.Itoa(id) + " was added at pool")
	}
	return nil
}

func (wp *WorkersPool) DeleteWorkers(count int) error {
	if count <= 0 || count > len(wp.workers) {
		return errors.New("Bad value of count: " + strconv.Itoa(count))
	}

	for i := 0; i < count; i++ {
		wp.workers[len(wp.workers)-i-1].done <- struct{}{}
	}
	wp.workers = wp.workers[:len(wp.workers)-count]
	return nil
}

func (wp *WorkersPool) Stop() {
	close(wp.jobsCh)
	close(wp.done)
	wp.wg.Wait()
	log.Println("[DEBUG] Workers-pool was stopped")
}
