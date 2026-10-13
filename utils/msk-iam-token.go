package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

// OAuthBearerTokenSetter is implemented by *kafka.Consumer and *kafka.Producer
type OAuthBearerTokenSetter interface {
	SetOAuthBearerToken(oauthBearerToken kafka.OAuthBearerToken) error
	SetOAuthBearerTokenFailure(errstr string) error
}

// generateMskIamToken signs the token with the default AWS credentials chain (EKS Pod Identity, IRSA, env vars, profile...)
var generateMskIamToken = signer.GenerateAuthToken

// mskBrokerRegion extracts the region of an MSK broker (e.g. b-1.cluster.xxxx.c2.kafka.sa-east-1.amazonaws.com:9098)
var mskBrokerRegion = regexp.MustCompile(`\.kafka\.([a-z0-9-]+)\.amazonaws\.com`)

const (
	mskIamTokenTimeout    = 10 * time.Second
	mskIamRetryInterval   = 10 * time.Second
	mskIamMinRefreshDelay = 5 * time.Second
)

// MskIamTokenRefresher sets the AWS_MSK_IAM token on a client and renews it before it expires.
// The renewal runs in its own goroutine because the consumer uses ReadMessage, which discards
// the OAuthBearerTokenRefresh events emitted by librdkafka.
type MskIamTokenRefresher struct {
	region   string
	client   OAuthBearerTokenSetter
	sendLog  func(types.LogEvent)
	instance string
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// MskIamRegion resolves the region used to sign the token: AWS_REGION, AWS_DEFAULT_REGION or the bootstrap servers hostname
func MskIamRegion(kafkaConfigMap kafka.ConfigMap) (string, error) {
	for _, envVar := range []string{"AWS_REGION", "AWS_DEFAULT_REGION"} {
		if region := os.Getenv(envVar); region != "" {
			return region, nil
		}
	}
	bootstrapServers, _ := kafkaConfigMap["bootstrap.servers"].(string)
	if match := mskBrokerRegion.FindStringSubmatch(bootstrapServers); match != nil {
		return match[1], nil
	}
	return "", errors.New("unable to resolve the AWS region for AWS_MSK_IAM (set AWS_REGION)")
}

// Start sets the first token synchronously (so the client can connect) and starts the renewal goroutine
func (r *MskIamTokenRefresher) Start(instance, region string, client OAuthBearerTokenSetter, sendLog func(types.LogEvent)) error {
	if client == nil || region == "" || sendLog == nil {
		return errors.New("missing params")
	}
	r.instance = instance
	r.region = region
	r.client = client
	r.sendLog = sendLog
	expiration, err := r.refresh()
	if err != nil {
		return fmt.Errorf("unable to generate the AWS_MSK_IAM token: %w", err)
	}
	r.stop = make(chan struct{})
	r.done = make(chan struct{})
	go r.loop(expiration)
	return nil
}

// Stop ends the renewal goroutine (safe to call more than once)
func (r *MskIamTokenRefresher) Stop() {
	if r.stop == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		<-r.done
	})
}

func (r *MskIamTokenRefresher) loop(expiration time.Time) {
	defer close(r.done)
	timer := time.NewTimer(nextRefreshDelay(expiration, time.Now()))
	defer timer.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-timer.C:
			newExpiration, err := r.refresh()
			if err != nil {
				r.sendLog(types.LogEvent{
					InstanceName: r.instance,
					Type:         "Error",
					Tag:          "AUTH",
					Message:      "AWS_MSK_IAM token refresh failed: " + err.Error(),
				})
				timer.Reset(mskIamRetryInterval)
				continue
			}
			timer.Reset(nextRefreshDelay(newExpiration, time.Now()))
		}
	}
}

func (r *MskIamTokenRefresher) refresh() (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mskIamTokenTimeout)
	defer cancel()
	token, expirationMs, err := generateMskIamToken(ctx, r.region)
	if err != nil {
		_ = r.client.SetOAuthBearerTokenFailure(err.Error())
		return time.Time{}, err
	}
	expiration := time.UnixMilli(expirationMs)
	err = r.client.SetOAuthBearerToken(kafka.OAuthBearerToken{
		TokenValue: token,
		Expiration: expiration,
	})
	if err != nil {
		_ = r.client.SetOAuthBearerTokenFailure(err.Error())
		return time.Time{}, err
	}
	return expiration, nil
}

// nextRefreshDelay renews at 80% of the remaining lifetime (the token lasts 15 minutes)
func nextRefreshDelay(expiration, now time.Time) time.Duration {
	delay := expiration.Sub(now) * 8 / 10
	if delay < mskIamMinRefreshDelay {
		return mskIamMinRefreshDelay
	}
	return delay
}
