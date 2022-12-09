package plugintest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/models"
	pluginLib "github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/coreplugin"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/provider"
	"github.com/grafana/grafana/pkg/plugins/config"
	"github.com/grafana/grafana/pkg/plugins/manager/client"
	"github.com/grafana/grafana/pkg/plugins/manager/fakes"
	"github.com/grafana/grafana/pkg/plugins/manager/loader"
	"github.com/grafana/grafana/pkg/plugins/manager/registry"
	"github.com/grafana/grafana/pkg/plugins/manager/signature"
	"github.com/grafana/grafana/pkg/plugins/manager/store"
	"github.com/grafana/grafana/pkg/services/auth/jwt"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/licensing"
	"github.com/grafana/grafana/pkg/services/plugins"
	"github.com/grafana/grafana/pkg/services/searchV2"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/azuremonitor"
	"github.com/grafana/grafana/pkg/tsdb/cloudmonitoring"
	"github.com/grafana/grafana/pkg/tsdb/cloudwatch"
	"github.com/grafana/grafana/pkg/tsdb/elasticsearch"
	"github.com/grafana/grafana/pkg/tsdb/grafanads"
	"github.com/grafana/grafana/pkg/tsdb/graphite"
	"github.com/grafana/grafana/pkg/tsdb/influxdb"
	"github.com/grafana/grafana/pkg/tsdb/loki"
	"github.com/grafana/grafana/pkg/tsdb/mssql"
	"github.com/grafana/grafana/pkg/tsdb/mysql"
	"github.com/grafana/grafana/pkg/tsdb/opentsdb"
	"github.com/grafana/grafana/pkg/tsdb/parca"
	"github.com/grafana/grafana/pkg/tsdb/phlare"
	"github.com/grafana/grafana/pkg/tsdb/postgres"
	"github.com/grafana/grafana/pkg/tsdb/prometheus"
	"github.com/grafana/grafana/pkg/tsdb/tempo"
	"github.com/grafana/grafana/pkg/tsdb/testdatasource"
)

func TestIntegrationPluginManager(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	staticRootPath, err := filepath.Abs("../../../../public/")
	require.NoError(t, err)

	bundledPluginsPath, err := filepath.Abs("../../../../plugins-bundled/internal")
	require.NoError(t, err)

	// We use the raw config here as it forms the basis for the setting.Provider implementation
	// The plugin manager also relies directly on the setting.Cfg struct to provide Grafana specific
	// properties such as the loading paths
	raw, err := ini.Load([]byte(`
		app_mode = production

		[plugin.test-app]
		path=../../../plugins/manager/testdata/test-app

		[plugin.test-panel]
		not=included
		`),
	)
	require.NoError(t, err)

	cfg := &setting.Cfg{
		Raw:                    raw,
		StaticRootPath:         staticRootPath,
		BundledPluginsPath:     bundledPluginsPath,
		Azure:                  &azsettings.AzureSettings{},
		IsFeatureToggleEnabled: func(_ string) bool { return false },
	}

	tracer := tracing.InitializeTracerForTest()
	features := featuremgmt.WithFeatures()

	hcp := httpclient.NewProvider()
	am := azuremonitor.ProvideService(cfg, hcp, tracer)
	cw := cloudwatch.ProvideService(cfg, hcp, features)
	cm := cloudmonitoring.ProvideService(hcp, tracer)
	es := elasticsearch.ProvideService(hcp)
	grap := graphite.ProvideService(hcp, tracer)
	idb := influxdb.ProvideService(hcp)
	lk := loki.ProvideService(hcp, features, tracer)
	otsdb := opentsdb.ProvideService(hcp)
	pr := prometheus.ProvideService(hcp, cfg, features, tracer)
	tmpo := tempo.ProvideService(hcp)
	td := testdatasource.ProvideService(cfg, features)
	pg := postgres.ProvideService(cfg)
	my := mysql.ProvideService(cfg, hcp)
	ms := mssql.ProvideService(cfg)
	sv2 := searchV2.ProvideService(cfg, db.InitTestDB(t), nil, nil, tracer, features, nil, nil, nil)
	graf := grafanads.ProvideService(sv2, nil)
	phlare := phlare.ProvideService(hcp)
	parca := parca.ProvideService(hcp)

	coreRegistry := coreplugin.ProvideCoreRegistry(am, cw, cm, es, grap, idb, lk, otsdb, pr, tmpo, td, pg, my, ms, graf, phlare, parca)

	pCfg := config.ProvideConfig(setting.ProvideProvider(cfg), cfg)
	reg := registry.ProvideService()
	l := loader.ProvideService(pCfg, &licensing.OSSLicensingService{Cfg: cfg}, signature.NewUnsignedAuthorizer(pCfg),
		reg, provider.ProvideService(coreRegistry), fakes.NewFakeRoleRegistry())
	ps, err := store.ProvideService(cfg, pCfg, reg, l)
	require.NoError(t, err)
	rr := plugins.ProvideRouteResolver(ps)
	require.NoError(t, err)

	ctx := context.Background()
	err = ps.Run(ctx)
	require.NoError(t, err)

	verifyCorePluginCatalogue(t, ctx, ps)
	verifyBundledPlugins(t, ctx, ps, reg)
	verifyPluginStaticRoutes(t, ctx, rr, reg)
	verifyBackendProcesses(t, reg.Plugins(ctx))
	verifyPluginQuery(t, ctx, client.ProvideService(reg, pCfg, &fakeJWTAuth{}))
}

func verifyPluginQuery(t *testing.T, ctx context.Context, c plugins.Client) {
	now := time.Unix(1661420870, 0)
	req := &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			PluginID: "testdata",
		},
		Queries: []backend.DataQuery{
			{
				RefID: "A",
				TimeRange: backend.TimeRange{
					From: now.Add(-5 * time.Minute),
					To:   now,
				},
				JSON: json.RawMessage(`{"scenarioId":"csv_metric_values","stringInput":"1,20,90,30,5,0"}`),
			},
		},
	}

	resp, err := c.QueryData(ctx, req)
	require.NoError(t, err)
	payload, err := resp.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{"results":{"A":{"frames":[{"schema":{"refId":"A","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"A-series","type":"number","typeInfo":{"frame":"int64","nullable":true}}]},"data":{"values":[[1661420570000,1661420630000,1661420690000,1661420750000,1661420810000,1661420870000],[1,20,90,30,5,0]]}}],"status":200}}}`, string(payload))
}

func verifyCorePluginCatalogue(t *testing.T, ctx context.Context, ps *store.Service) {
	t.Helper()

	expPanels := map[string]struct{}{
		"alertGroups":    {},
		"alertlist":      {},
		"annolist":       {},
		"barchart":       {},
		"bargauge":       {},
		"canvas":         {},
		"dashlist":       {},
		"debug":          {},
		"gauge":          {},
		"geomap":         {},
		"gettingstarted": {},
		"graph":          {},
		"heatmap":        {},
		"histogram":      {},
		"icon":           {},
		"live":           {},
		"logs":           {},
		"candlestick":    {},
		"news":           {},
		"nodeGraph":      {},
		"flamegraph":     {},
		"traces":         {},
		"piechart":       {},
		"stat":           {},
		"state-timeline": {},
		"status-history": {},
		"table":          {},
		"table-old":      {},
		"text":           {},
		"timeseries":     {},
		"welcome":        {},
		"xychart":        {},
	}

	expDataSources := map[string]struct{}{
		"cloudwatch":                       {},
		"stackdriver":                      {},
		"grafana-azure-monitor-datasource": {},
		"elasticsearch":                    {},
		"graphite":                         {},
		"influxdb":                         {},
		"loki":                             {},
		"opentsdb":                         {},
		"prometheus":                       {},
		"tempo":                            {},
		"testdata":                         {},
		"postgres":                         {},
		"mysql":                            {},
		"mssql":                            {},
		"grafana":                          {},
		"alertmanager":                     {},
		"dashboard":                        {},
		"input":                            {},
		"jaeger":                           {},
		"mixed":                            {},
		"zipkin":                           {},
		"phlare":                           {},
		"parca":                            {},
	}

	expApps := map[string]struct{}{
		"test-app": {},
	}

	panels := ps.Plugins(ctx, pluginLib.Panel)
	require.Equal(t, len(expPanels), len(panels))
	for _, p := range panels {
		p, exists := ps.Plugin(ctx, p.ID)
		require.NotEqual(t, plugins.PluginDTO{}, p)
		require.True(t, exists)
		require.Contains(t, expPanels, p.ID)
	}

	dataSources := ps.Plugins(ctx, pluginLib.DataSource)
	require.Equal(t, len(expDataSources), len(dataSources))
	for _, ds := range dataSources {
		p, exists := ps.Plugin(ctx, ds.ID)
		require.NotEqual(t, plugins.PluginDTO{}, p)
		require.True(t, exists)
		require.Contains(t, expDataSources, ds.ID)
	}

	apps := ps.Plugins(ctx, pluginLib.App)
	require.Equal(t, len(expApps), len(apps))
	for _, app := range apps {
		p, exists := ps.Plugin(ctx, app.ID)
		require.True(t, exists)
		require.NotNil(t, p)
		require.Contains(t, expApps, app.ID)
	}

	require.Equal(t, len(expPanels)+len(expDataSources)+len(expApps), len(ps.Plugins(ctx)))
}

func verifyBundledPlugins(t *testing.T, ctx context.Context, ps *store.Service, reg registry.Service) {
	t.Helper()

	dsPlugins := make(map[string]struct{})
	for _, p := range ps.Plugins(ctx, pluginLib.DataSource) {
		dsPlugins[p.ID] = struct{}{}
	}

	inputPlugin, exists := ps.Plugin(ctx, "input")
	require.True(t, exists)
	require.NotEqual(t, plugins.PluginDTO{}, inputPlugin)
	require.NotNil(t, dsPlugins["input"])

	intInputPlugin, exists := reg.Plugin(ctx, "input")
	require.True(t, exists)

	pluginRoutes := make(map[string]*pluginLib.StaticRoute)
	for _, r := range ps.Routes() {
		pluginRoutes[r.PluginID] = r
	}

	for _, pluginID := range []string{"input"} {
		require.Contains(t, pluginRoutes, pluginID)
		require.True(t, strings.HasPrefix(pluginRoutes[pluginID].Directory, intInputPlugin.PluginDir))
	}
}

func verifyPluginStaticRoutes(t *testing.T, ctx context.Context, rr plugins.StaticRouteResolver, reg registry.Service) {
	routes := make(map[string]*plugins.StaticRoute)
	for _, route := range rr.Routes(ctx) {
		routes[route.PluginID] = route
	}

	require.Len(t, routes, 2)

	inputPlugin, _ := reg.Plugin(ctx, "input")
	require.NotNil(t, routes["input"])
	require.Equal(t, routes["input"].Directory, inputPlugin.PluginDir)

	testAppPlugin, _ := reg.Plugin(ctx, "test-app")
	require.Contains(t, routes, "test-app")
	require.Equal(t, routes["test-app"].Directory, testAppPlugin.PluginDir)
}

func verifyBackendProcesses(t *testing.T, ps []*pluginLib.Plugin) {
	for _, p := range ps {
		if p.Backend {
			pc, exists := p.Client()
			require.True(t, exists)
			require.NotNil(t, pc)
		}
	}
}

type fakeJWTAuth struct {
	jwt.PluginAuthService
}

func (f *fakeJWTAuth) IsEnabled() bool {
	return true
}

func (f *fakeJWTAuth) Generate(*user.SignedInUser, string) (string, error) {
	return "", nil
}

func (f *fakeJWTAuth) Verify(_ context.Context, _ string) (models.JWTClaims, error) {
	return models.JWTClaims{}, nil
}
