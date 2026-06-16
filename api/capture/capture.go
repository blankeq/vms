package capture

import (
	"context"
	"log"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type FrameSubscriber chan gocv.Mat

type CaptureInstance struct {
	capture     *gocv.VideoCapture
	subscribers map[FrameSubscriber]struct{}
	mtx         sync.RWMutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

type CaptureManager struct {
	mu        sync.Mutex
	instances map[int]*CaptureInstance
}

func NewCaptureManager() *CaptureManager {
	return &CaptureManager{
		instances: make(map[int]*CaptureInstance),
	}
}

var Manager = NewCaptureManager()

func (m *CaptureManager) Subscribe(cameraID int, rtspLink string) (FrameSubscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[cameraID]
	if !ok {
		cap, err := gocv.OpenVideoCapture(rtspLink)
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithCancel(context.Background())
		instance = &CaptureInstance{
			capture:     cap,
			subscribers: make(map[FrameSubscriber]struct{}),
			cancel:      cancel,
		}
		m.instances[cameraID] = instance
		instance.wg.Add(1)

		go instance.readLoop(ctx)

		log.Printf("Capture started for camera %d", cameraID)
	}

	ch := make(FrameSubscriber, 2)

	instance.mtx.Lock()
	instance.subscribers[ch] = struct{}{}
	instance.mtx.Unlock()

	return ch, nil
}

func (m *CaptureManager) Unsubscribe(cameraID int, ch FrameSubscriber) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[cameraID]
	if !ok {
		return ErrInstanceNotExist
	}

	instance.mtx.Lock()
	if _, exists := instance.subscribers[ch]; exists {
		delete(instance.subscribers, ch)
		close(ch)
	}
	left := len(instance.subscribers)
	instance.mtx.Unlock()

	if left == 0 {
		instance.cancel()
		instance.wg.Wait()
		instance.capture.Close()
		delete(m.instances, cameraID)
		log.Printf("Capture stopped for camera %d", cameraID)
	}

	return nil
}

func (inst *CaptureInstance) readLoop(ctx context.Context) {
	defer inst.wg.Done()

	img := gocv.NewMat()
	defer img.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := inst.capture.Read(&img); !ok {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if img.Empty() {
				continue
			}

			inst.mtx.RLock()
			for ch := range inst.subscribers {
				cloned := img.Clone()

				select {
				case ch <- cloned:
				default:
					cloned.Close()
					log.Println("Frame subscriber slow, dropping frame...")
				}
			}
			inst.mtx.RUnlock()
		}
	}
}
