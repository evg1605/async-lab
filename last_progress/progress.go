package last_progress

import "io"

type Producer interface {
	io.Closer
	SetLastProgress(v interface{})
}

type Consumer interface {
	GetLastProgress() <-chan interface{}
}

type Progress interface {
	Producer
	Consumer
}

type progress struct {
	getCh   chan interface{}
	setCh   chan interface{}
	closeCh chan struct{}
}

func NewProgress() Progress {
	pr := &progress{
		getCh:   make(chan interface{}),
		setCh:   make(chan interface{}),
		closeCh: make(chan struct{}),
	}
	go pr.progressLoop()
	return pr
}

func (pr *progress) GetLastProgress() <-chan interface{} {
	return pr.getCh
}

func (pr *progress) SetLastProgress(v interface{}) {
	select {
	case pr.setCh <- v:
	case <-pr.closeCh:
	}
}

func (pr *progress) Close() error {
	close(pr.closeCh)
	return nil
}

func (pr *progress) progressLoop() {
	var lastVal interface{}
	var getCh chan<- interface{}
	for {
		select {
		case getCh <- lastVal:
			getCh = nil
			lastVal = nil
		case lastVal = <-pr.setCh:
			getCh = pr.getCh
		case <-pr.closeCh:
			return
		}
	}
}
