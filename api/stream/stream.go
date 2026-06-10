package stream

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

type StreamManager struct {
	mtx           sync.Mutex
	activeStreams map[int]StreamRegistry
}

type StreamRegistry struct {
	Stream *mjpeg.Stream
	Cancel context.CancelFunc
}

func NewStreamRegistry(stream *mjpeg.Stream, cancel context.CancelFunc) StreamRegistry {
	return StreamRegistry{
		Stream: stream,
		Cancel: cancel,
	}
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		activeStreams: make(map[int]StreamRegistry, 0),
	}
}

var Manager = NewStreamManager()

func (s *StreamManager) StartStream(cameraId int, rtspLink string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.activeStreams[cameraId]
	if ok {
		return ErrStreamAlreadyExist
	}

	video, err := gocv.OpenVideoCapture(rtspLink)
	if err != nil {
		video.Close()
		return ErrTryingToOpenVideoCapture
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := mjpeg.NewStreamWithContext(ctx)

	registry := NewStreamRegistry(stream, cancel)
	s.activeStreams[cameraId] = registry

	go s.streamLoop(ctx, video, stream)

	fmt.Println("Camera", cameraId, "started streaming")

	return nil
}

func (s *StreamManager) streamLoop(ctx context.Context, video *gocv.VideoCapture, stream *mjpeg.Stream) {
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := video.Read(&img); !ok {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			if img.Empty() {
				continue
			}

			buf, err := gocv.IMEncode(".jpg", img)
			if err == nil {
				stream.UpdateJPEG(buf.GetBytes())
				buf.Close()
			}
		}
	}
}

func (s *StreamManager) StopStream(cameraId int) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return ErrStreamNotExist
	}

	registry.Cancel()
	delete(s.activeStreams, cameraId)

	fmt.Println("Camera", cameraId, "stopped streaming")

	return nil
}

func (s *StreamManager) GetStream(cameraId int) (*mjpeg.Stream, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return &mjpeg.Stream{}, ErrStreamNotExist
	}

	return registry.Stream, nil
}

func (s *StreamManager) StopAll() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for cameraId, registry := range s.activeStreams {
		registry.Cancel()
		delete(s.activeStreams, cameraId)
	}

	log.Println("Stopped all streams...")
}
