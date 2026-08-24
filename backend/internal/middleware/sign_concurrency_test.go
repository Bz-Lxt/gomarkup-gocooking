package middleware_test

import (
	"fmt"
	"sync"
	"testing"

	"gocooking/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
)

func TestConcurrentSignKeepsCallerClaims(t *testing.T) {
	const (
		workers = 24
		rounds  = 100
		secret  = "concurrent-signing-test-secret"
	)

	start := make(chan struct{})
	failures := make(chan string, workers)
	var wg sync.WaitGroup
	for i := 1; i <= workers; i++ {
		userID := uint(i)
		username := fmt.Sprintf("user-%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for n := 0; n < rounds; n++ {
				tokenString, err := middleware.Sign(secret, userID, username)
				if err != nil {
					failures <- fmt.Sprintf("sign user %d: %v", userID, err)
					return
				}

				claims := &middleware.Claims{}
				token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
					return []byte(secret), nil
				})
				if err != nil || !token.Valid {
					failures <- fmt.Sprintf("parse token for user %d: %v", userID, err)
					return
				}
				if claims.UserID != userID || claims.Username != username {
					failures <- fmt.Sprintf(
						"token for user %d/%q contains user %d/%q",
						userID, username, claims.UserID, claims.Username,
					)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(failures)
	if failure, ok := <-failures; ok {
		t.Fatal(failure)
	}
}
