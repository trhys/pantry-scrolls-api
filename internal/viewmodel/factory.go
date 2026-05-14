package viewmodel

import (
	"github.com/trhys/Recipe-Repo-2/internal/database"
)

type VMFactory struct{
	DB 	*database.Queries
	S3cdn	string
}
