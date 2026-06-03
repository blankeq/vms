package stream

import (
	"sync"

	"github.com/hybridgroup/mjpeg"
)

type StreamManager struct {
	mtx           sync.Mutex
	activeStreams map[int]*mjpeg.Stream
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		activeStreams: make(map[int]*mjpeg.Stream, 0),
	}
}

var Manager = NewStreamManager()

func (s *StreamManager) StartStream(cameraId int, stream *mjpeg.Stream) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.activeStreams[cameraId]
	if ok {
		return ErrStreamAlreadyExist
	}

	s.activeStreams[cameraId] = stream

	return nil
}

func (s *StreamManager) StopStream(cameraId int) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.activeStreams[cameraId]
	if !ok {
		return ErrStreamNotExist
	}

	delete(s.activeStreams, cameraId)

	return nil
}

func (s *StreamManager) GetStream(cameraId int) (*mjpeg.Stream, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	stream, ok := s.activeStreams[cameraId]
	if !ok {
		return &mjpeg.Stream{}, ErrStreamNotExist
	}

	return stream, nil
}
