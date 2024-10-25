package byterange

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/eikenb/pipeat"
	"github.com/pkg/errors"
)

type dispatcher struct {
	fileSize  int64
	rangeSize int64
	todos     JobQueue
	workers   chan worker
	resp      chan response
	writer    *pipeat.PipeWriterAt
	reader    *pipeat.PipeReaderAt
	backoff   *backoff
}

type worker struct {
	// c *http.Client
	e string
}

type response struct {
	offset int64
	data   []byte
}

type job struct {
	index int
	start int64
	end   int64
	retry int
}

type backoff struct {
	minDelay time.Duration
	maxDelay time.Duration
}

func (b *backoff) next(attempt int) time.Duration {
	if attempt < 0 {
		return b.minDelay
	}

	minf := float64(b.minDelay)
	durf := minf * math.Pow(1.5, float64(attempt))
	durf = durf + rand.Float64()*minf

	delay := time.Duration(durf)
	if delay > b.maxDelay {
		return b.maxDelay
	}

	return delay
}

func (d *dispatcher) generateJobs() {
	count := int64(math.Ceil(float64(d.fileSize) / float64(d.rangeSize)))
	for i := int64(0); i < count; i++ {
		start := i * d.rangeSize
		end := (i + 1) * d.rangeSize

		if end > d.fileSize {
			end = d.fileSize
		}

		newJob := &job{
			index: int(i),
			start: start,
			end:   end,
		}

		d.todos.Push(newJob)
	}
}

func (d *dispatcher) run(ctx context.Context) {
	d.generateJobs()
	d.writeData(ctx)

	var (
		counter  int64
		finished = make(chan int64, 1)
	)

	// go func() {
	for {
		select {
		case w := <-d.workers:
			go func() {
				j, ok := d.todos.Pop()
				if !ok {
					d.workers <- w
					return
				}

				data, err := d.fetch(ctx, w, j)
				if err != nil {
					errMsg := fmt.Sprintf("pull data failed : %v", err)
					if j.retry > 0 {
						log.Errorf("pull data failed (retries: %d): %v", j.retry, err)
						<-time.After(d.backoff.next(j.retry))
					}

					log.Warnf(errMsg)

					j.retry++
					d.todos.PushFront(j)
					d.workers <- w
					return
				}

				dataLen := j.end - j.start

				if int64(len(data)) < dataLen {
					log.Errorf("unexpected data size, want %d got %d", dataLen, len(data))
					d.todos.PushFront(j)
					d.workers <- w
					return
				}

				d.workers <- w
				d.resp <- response{
					data:   data[:dataLen],
					offset: j.start,
				}
				finished <- dataLen
			}()
		case size := <-finished:
			counter += size
			if counter >= d.fileSize {
				return
			}
		case <-ctx.Done():
			return
		}
	}
	// }()

	// return
}

func (d *dispatcher) writeData(ctx context.Context) {
	go func() {
		defer d.finally()

		var count int64
		for {
			select {
			case r := <-d.resp:
				_, err := d.writer.WriteAt(r.data, r.offset)
				if err != nil {
					log.Errorf("write data failed: %v", err)
					continue
				}

				count += int64(len(r.data))
				if count >= d.fileSize {
					return
				}
			case <-ctx.Done():
				return
			}
		}

	}()
}

func (d *dispatcher) fetch(ctx context.Context, w worker, j *job) ([]byte, error) {
	startTime := time.Now()
	req, err := http.NewRequest("GET", w.e, nil)
	if err != nil {
		return nil, errors.Errorf("new request failed: %v", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", j.start, j.end))
	// resp, err := w.c.Do(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("fetch failed: %v", err)
	}

	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download chunk: %d-%d, status code: %d", j.start, j.end, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("read data failed: %v", err)
	}

	elapsed := time.Since(startTime)
	log.Infof("Chunk: %fs, Link: %s", elapsed.Seconds(), w.e)

	return data, nil
}

func (d *dispatcher) finally() {
	if err := d.writer.Close(); err != nil {
		log.Errorf("close write failed: %v", err)
	}
}
