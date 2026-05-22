package server

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trhys/Recipe-Repo-2/internal/database"
	"github.com/trhys/Recipe-Repo-2/internal/viewmodel"

)

type apiConfig struct {
	db		*database.Queries
	platform	string
	secret		string
	jwtDuration	time.Duration
	s3client	*s3.Client
	s3bucket	string
	s3region	string
	s3cdn		string
	imagePlaceholder string

	vmf		viewmodel.VMFactory
}
