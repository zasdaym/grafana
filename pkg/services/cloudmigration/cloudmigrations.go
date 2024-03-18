package cloudmigration

import (
	"context"
)

type Service interface {
	MigrateDatasources(context.Context, *MigrateDatasourcesRequest) (*MigrateDatasourcesResponse, error)
	CreateAccessToken(ctx context.Context) (CreateAccessTokenResponse, error)
}
