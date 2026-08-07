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
	ReaperInterval   time.Duration
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

// db transaction helper with rollback
func (cfg *ApiConfig) withTx(ctx context.Context, fn func(qtx *database.Queries) error) error {
	tx, err := cfg.DBConn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	qtx := cfg.DB.WithTx(tx)

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(qtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
