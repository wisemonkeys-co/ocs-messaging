package utils

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

type fakeTokenClient struct {
	mu       sync.Mutex
	tokens   []kafka.OAuthBearerToken
	failures []string
}

func (f *fakeTokenClient) SetOAuthBearerToken(token kafka.OAuthBearerToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens = append(f.tokens, token)
	return nil
}

func (f *fakeTokenClient) SetOAuthBearerTokenFailure(errstr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures = append(f.failures, errstr)
	return nil
}

func (f *fakeTokenClient) count() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.tokens), len(f.failures)
}

func mockTokenGenerator(t *testing.T, generator func(ctx context.Context, region string) (string, int64, error)) {
	original := generateMskIamToken
	generateMskIamToken = generator
	t.Cleanup(func() { generateMskIamToken = original })
}

func TestMskIamRegion(t *testing.T) {
	brokers := kafka.ConfigMap{"bootstrap.servers": "b-1.wise.abc123.c2.kafka.sa-east-1.amazonaws.com:9098,b-2.wise.abc123.c2.kafka.sa-east-1.amazonaws.com:9098"}
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	if region, err := MskIamRegion(brokers); err != nil || region != "sa-east-1" {
		t.Errorf("region from bootstrap servers: got %q, %v", region, err)
	}
	if _, err := MskIamRegion(kafka.ConfigMap{"bootstrap.servers": "localhost:9092"}); err == nil {
		t.Error("expected error without region")
	}
	t.Setenv("AWS_REGION", "us-east-1")
	if region, _ := MskIamRegion(brokers); region != "us-east-1" {
		t.Errorf("AWS_REGION should take precedence, got %q", region)
	}
}

func TestMskIamTokenRefresherSetsAndRenewsToken(t *testing.T) {
	var regions []string
	var mu sync.Mutex
	mockTokenGenerator(t, func(ctx context.Context, region string) (string, int64, error) {
		mu.Lock()
		regions = append(regions, region)
		mu.Unlock()
		// Short lifetime: the next renewal happens after mskIamMinRefreshDelay
		return "token", time.Now().Add(time.Second).UnixMilli(), nil
	})
	client := &fakeTokenClient{}
	refresher := MskIamTokenRefresher{}
	if err := refresher.Start("test", "sa-east-1", client, func(types.LogEvent) {}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens, _ := client.count(); tokens != 1 {
		t.Fatalf("first token must be set synchronously, got %d", tokens)
	}
	time.Sleep(mskIamMinRefreshDelay + time.Second)
	refresher.Stop()
	refresher.Stop()
	tokens, failures := client.count()
	if tokens < 2 || failures != 0 {
		t.Errorf("expected renewal: tokens=%d failures=%d", tokens, failures)
	}
	mu.Lock()
	defer mu.Unlock()
	if regions[0] != "sa-east-1" {
		t.Errorf("unexpected region %q", regions[0])
	}
}

func TestMskIamTokenRefresherFailure(t *testing.T) {
	mockTokenGenerator(t, func(ctx context.Context, region string) (string, int64, error) {
		return "", 0, errors.New("no credentials")
	})
	client := &fakeTokenClient{}
	refresher := MskIamTokenRefresher{}
	if err := refresher.Start("test", "sa-east-1", client, func(types.LogEvent) {}); err == nil {
		t.Fatal("expected error when the first token cannot be generated")
	}
	if _, failures := client.count(); failures != 1 {
		t.Errorf("expected SetOAuthBearerTokenFailure, got %d", failures)
	}
	refresher.Stop()
}

func TestNextRefreshDelay(t *testing.T) {
	now := time.Now()
	if got := nextRefreshDelay(now.Add(15*time.Minute), now); got != 12*time.Minute {
		t.Errorf("expected 12m, got %v", got)
	}
	if got := nextRefreshDelay(now.Add(-time.Minute), now); got != mskIamMinRefreshDelay {
		t.Errorf("expected minimum delay, got %v", got)
	}
}
