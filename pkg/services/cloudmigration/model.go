package cloudmigration

import (
	"github.com/grafana/grafana/pkg/util/errutil"
)

var (
	ErrInternalNotImplementedError = errutil.Internal("cloudmigrations.notImplemented", errutil.WithPublicMessage("Internal server error"))
	ErrFeatureDisabledError        = errutil.Internal("cloudmigrations.disabled", errutil.WithPublicMessage("Cloud migrations are disabled on this instance"))
)

type MigrateDatasourcesRequest struct {
	MigrateToPDC       bool
	MigrateCredentials bool
}

type MigrateDatasourcesResponse struct {
	DatasourcesMigrated int
}

type MigrateDatasourcesRequestDTO struct {
	MigrateToPDC       bool `json:"migrateToPDC"`
	MigrateCredentials bool `json:"migrateCredentials"`
}

type MigrateDatasourcesResponseDTO struct {
	DatasourcesMigrated int `json:"datasourcesMigrated"`
}

type CreateAccessTokenResponse struct {
	Token string
}

type CreateAccessTokenResponseDTO struct {
	Token string `json:"token"`
}
