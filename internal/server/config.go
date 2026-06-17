package server

import (
	"context"
	"database/sql"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"
)

type ApiConfig struct {
	DB               *database.Queries
	DBConn           *sql.DB
	Secret           string
	JwtDuration      time.Duration
	SESClient        EmailClient
	S3client         *s3.Client
	S3bucket         string
	S3region         string
	S3cdn            string
	ImagePlaceholder string

	Vmf   viewmodel.VMFactory
	Root  adminCredentials
	React string
}

type adminCredentials struct {
	Email string
	Pass  string
}

type EmailClient interface {
	SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}
