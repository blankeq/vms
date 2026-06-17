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
	mtx       sync.Mutex
	instances map[int]*CaptureInstance
}

func NewCaptureManager() *CaptureManager {
	return &CaptureManager{
		instances: make(map[int]*CaptureInstance),
	}
}

var Manager = NewCaptureManager()

func (cm *CaptureManager) Subscribe(cameraId int, rtspLink string) (FrameSubscriber, error) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()

	instance, ok := cm.instances[cameraId]
	if !ok {
		capture, err := gocv.OpenVideoCapture(rtspLink)
		if err != nil {
			return nil, ErrTryingToOpenVideoCapture
		}

		ctx, cancel := context.WithCancel(context.Background())

		instance = &CaptureInstance{
			capture:     capture,
			subscribers: make(map[FrameSubscriber]struct{}),
			cancel:      cancel,
		}
		cm.instances[cameraId] = instance
		instance.wg.Add(1)

		go instance.readLoop(ctx, cameraId)

		log.Printf("[Camera %d] Capture started...", cameraId)
	}

	ch := make(FrameSubscriber, 2)

	instance.mtx.Lock()
	instance.subscribers[ch] = struct{}{}
	instance.mtx.Unlock()

	return ch, nil
}

func (cm *CaptureManager) Unsubscribe(cameraId int, ch FrameSubscriber) error {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()

	instance, ok := cm.instances[cameraId]
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
		delete(cm.instances, cameraId)

		log.Printf("[Camera %d] Capture stopped...", cameraId)
	}

	return nil
}

func (ci *CaptureInstance) readLoop(ctx context.Context, cameraId int) {
	defer ci.wg.Done()

	img := gocv.NewMat()
	defer img.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := ci.capture.Read(&img); !ok {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if img.Empty() {
				continue
			}

			ci.mtx.RLock()
			for ch := range ci.subscribers {
				cloned := img.Clone()

				select {
				case ch <- cloned:
				default:
					cloned.Close()
					log.Printf("[Camera %d] Frame subscriber slow, dropping frame...", cameraId)
				}
			}
			ci.mtx.RUnlock()
		}
	}
}
