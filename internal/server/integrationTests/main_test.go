package server_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../.env"); err != nil {
		log.Fatal("ENV not found")
	}

	os.Exit(m.Run())
}

// MockSESClient intercepts SendEmail requests to keep integration tests local
type MockSESClient struct {
	LastToAddress string
	CallCount     int
}

func (m *MockSESClient) SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
	m.CallCount++

	if params.Destination != nil && len(params.Destination.ToAddresses) > 0 {
		m.LastToAddress = params.Destination.ToAddresses[0]
	}

	id := "mock-message-id-12345"
	return &ses.SendEmailOutput{
		MessageId: &id,
	}, nil
}
