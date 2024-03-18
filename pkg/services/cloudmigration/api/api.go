package api

import (
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/middleware"
	"github.com/grafana/grafana/pkg/services/cloudmigration"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/web"
)

type MigrationAPI struct {
	cloudMigrationsService cloudmigration.Service
	routeRegister          routing.RouteRegister
	log                    log.Logger
	tracer                 tracing.Tracer
}

func RegisterApi(
	rr routing.RouteRegister,
	cms cloudmigration.Service,
	tracer tracing.Tracer,
) *MigrationAPI {
	api := &MigrationAPI{
		log:                    log.New("cloudmigrations.api"),
		routeRegister:          rr,
		cloudMigrationsService: cms,
		tracer:                 tracer,
	}
	api.registerEndpoints()
	return api
}

// RegisterAPIEndpoints Registers Endpoints on Grafana Router
func (api *MigrationAPI) registerEndpoints() {
	api.routeRegister.Group("/api/cloudmigrations", func(apiRoute routing.RouteRegister) {
		apiRoute.Post(
			"/migrate_datasources",
			routing.Wrap(api.MigrateDatasources),
		)
		apiRoute.Post("/token", routing.Wrap(api.CreateAccessToken))
	}, middleware.ReqGrafanaAdmin)
}

func (api *MigrationAPI) MigrateDatasources(c *contextmodel.ReqContext) response.Response {
	var req cloudmigration.MigrateDatasourcesRequestDTO
	if err := web.Bind(c.Req, &req); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	resp, err := api.cloudMigrationsService.MigrateDatasources(c.Req.Context(), &cloudmigration.MigrateDatasourcesRequest{MigrateToPDC: req.MigrateToPDC, MigrateCredentials: req.MigrateCredentials})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "data source migrations error", err)
	}

	return response.JSON(http.StatusOK, cloudmigration.MigrateDatasourcesResponseDTO{DatasourcesMigrated: resp.DatasourcesMigrated})
}

func (api *MigrationAPI) CreateAccessToken(c *contextmodel.ReqContext) response.Response {
	ctx, span := api.tracer.Start(c.Req.Context(), "MigrationAPI.CreateAccessToken")
	defer span.End()

	logger := api.log.FromContext(ctx)

	resp, err := api.cloudMigrationsService.CreateAccessToken(ctx)
	if err != nil {
		logger.Error("creating gcom access token", "err", err.Error())
		return response.Error(http.StatusInternalServerError, "creating gcom access token", err)
	}

	return response.JSON(http.StatusOK, cloudmigration.CreateAccessTokenResponseDTO(resp))
}
