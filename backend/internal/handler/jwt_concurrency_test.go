package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gocooking/internal/handler"
	"gocooking/internal/middleware"
	"gocooking/internal/service"
)

type gatedBody struct {
	started chan<- struct{}
	release <-chan struct{}
	body    *strings.Reader
	once    sync.Once
}

func (b *gatedBody) Read(p []byte) (int, error) {
	b.once.Do(func() {
		close(b.started)
		<-b.release
	})
	return b.body.Read(p)
}

func (b *gatedBody) Close() error { return nil }

func TestAuthenticatedRequestsDoNotBlockBehindSlowPeer(t *testing.T) {
	const secret = "test-secret"
	slowToken, err := middleware.Sign(secret, 101, "slow-user")
	if err != nil {
		t.Fatal(err)
	}
	fastToken, err := middleware.Sign(secret, 202, "fast-user")
	if err != nil {
		t.Fatal(err)
	}

	router := handler.NewRouter(handler.Deps{
		Planner: service.NewPlanner(nil),
		Secret:  secret,
	})

	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblockSlowRequest := func() {
		releaseOnce.Do(func() { close(release) })
	}
	defer unblockSlowRequest()
	slowBody := &gatedBody{
		started: started,
		release: release,
		body:    strings.NewReader("{"),
	}
	slowReq := httptest.NewRequest(http.MethodPost, "/api/v1/shopping-lists/generate", slowBody)
	slowReq.Header.Set("Authorization", "Bearer "+slowToken)
	slowReq.Header.Set("Content-Type", "application/json")
	slowRec := httptest.NewRecorder()
	slowDone := make(chan int, 1)
	go func() {
		router.ServeHTTP(slowRec, slowReq)
		slowDone <- slowRec.Code
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("slow request did not reach its handler")
	}

	fastReq := httptest.NewRequest(http.MethodGet, "/api/v1/meal-plan?week=not-a-date", nil)
	fastReq.Header.Set("Authorization", "Bearer "+fastToken)
	fastRec := httptest.NewRecorder()
	fastDone := make(chan int, 1)
	fastStarted := make(chan struct{})
	go func() {
		close(fastStarted)
		router.ServeHTTP(fastRec, fastReq)
		fastDone <- fastRec.Code
	}()
	<-fastStarted

	healthRec := httptest.NewRecorder()
	router.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthRec.Code, http.StatusOK)
	}

	fastStatus := 0
	blocked := false
	select {
	case fastStatus = <-fastDone:
	case <-time.After(500 * time.Millisecond):
		blocked = true
	}

	unblockSlowRequest()

	select {
	case status := <-slowDone:
		if status != http.StatusBadRequest {
			t.Fatalf("slow request status = %d, want %d", status, http.StatusBadRequest)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("slow request did not finish after its body was released")
	}

	if blocked {
		select {
		case fastStatus = <-fastDone:
		case <-time.After(2 * time.Second):
			t.Fatal("fast request remained blocked after the slow request finished")
		}
	}
	if fastStatus != http.StatusUnprocessableEntity {
		t.Fatalf("fast request status = %d, want %d", fastStatus, http.StatusUnprocessableEntity)
	}
	if blocked {
		t.Fatal("an unrelated authenticated request waited for the slow request")
	}
}
