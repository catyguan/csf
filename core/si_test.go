package core

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/catyguan/csf/core/corepb"
	"github.com/stretchr/testify/assert"
)

type debugService struct {
	resp *corepb.Response
	err  error
	wt   int
}

func newDebugService(r *corepb.Response, err error) *debugService {
	return &debugService{
		resp: r,
		err:  err,
	}
}

func (this *debugService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return req.IsExecuteType(), nil
}

func (this *debugService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	plog.Infof("ApplyRequest - %v", req)
	if this.wt > 0 {
		du := time.Duration(this.wt) * time.Millisecond
		plog.Infof("sleep - %v", du)
		time.Sleep(du)
	}
	return this.resp, this.err
}

func (this *debugService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	return this.err

}

func (this *debugService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	return this.err
}

var debugResponse = &corepb.Response{}

func TestSimple(t *testing.T) {
	ds := newDebugService(debugResponse, nil)
	si := NewSimpleServiceInvoker(ds)
	r, err := si.InvokeRequest(context.Background(), corepb.NewQueryRequest("tservice", "tpath", []byte("hello world")))
	assert.NoError(t, err)
	assert.Equal(t, r, debugResponse)
}

func TestAsyncChannelSend(t *testing.T) {
	ds := newDebugService(debugResponse, nil)
	ds.wt = 1000
	si := NewLockerServiceInvoker(ds, nil)
	si.AsyncChannelSend = true
	req := corepb.NewQueryRequest("tservice", "tpath", []byte("hello world"))
	r, err := si.SendRequest(context.Background(), &corepb.ChannelRequest{Request: req})
	plog.Infof("wait")
	resp := <-r
	plog.Infof("done")
	assert.NoError(t, err)
	assert.Equal(t, resp.Response, debugResponse)
}

func TestLocker(t *testing.T) {
	ds := newDebugService(debugResponse, nil)
	si := NewLockerServiceInvoker(ds, nil)

	wg := sync.WaitGroup{}
	wg.Add(2)

	f1 := func() {
		defer wg.Done()
		ctx := context.Background()
		req := corepb.NewQueryRequest("tservice", "tpath", []byte("query"))
		for i := 0; i < 10; i++ {
			req.Info.ID = uint64(i + 1)
			r, err := si.InvokeRequest(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, r, debugResponse)
		}
	}
	f2 := func() {
		defer wg.Done()
		ctx := context.Background()
		req := corepb.NewExecuteRequest("tservice", "tpath", []byte("execute"))
		for i := 0; i < 10; i++ {
			req.Info.ID = uint64(i + 1)
			r, err := si.InvokeRequest(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, r, debugResponse)
		}
	}
	go f1()
	go f2()

	wg.Wait()
}

func TestSingleTbhread(t *testing.T) {
	ds := newDebugService(debugResponse, nil)
	si := NewSingeThreadServiceInvoker(ds, 100)
	defer si.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	f1 := func() {
		defer wg.Done()
		ctx := context.Background()
		req := corepb.NewQueryRequest("tservice", "tpath", []byte("query"))
		for i := 0; i < 10; i++ {
			req.Info.ID = uint64(i + 1)
			r, err := si.InvokeRequest(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, r, debugResponse)
		}
	}
	f2 := func() {
		defer wg.Done()
		ctx := context.Background()
		req := corepb.NewExecuteRequest("tservice", "tpath", []byte("execute"))
		for i := 0; i < 11; i++ {
			req.Info.ID = uint64(i + 1)
			r, err := si.InvokeRequest(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, r, debugResponse)
		}
	}
	go f1()
	go f2()

	wg.Wait()
}

func TestContextTimeout(t *testing.T) {
	ds := newDebugService(debugResponse, nil)
	ds.wt = 2000
	si := NewSingeThreadServiceInvoker(ds, 100)
	defer si.Close()

	ctx := context.Background()
	ctx2, _ := context.WithTimeout(ctx, 1*time.Second)
	req := corepb.NewQueryRequest("tservice", "tpath", []byte("query"))
	r, err := DoSendRequest(si, ctx2, &corepb.ChannelRequest{Request: req})
	assert.Error(t, err)
	plog.Infof("timeout - %v", err)
	assert.Nil(t, r)
}
