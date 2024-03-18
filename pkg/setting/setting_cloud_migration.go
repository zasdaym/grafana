package setting

import (
	"time"
)

type CloudMigrationSettings struct {
	IsTarget                  bool
	Region                    string
	GcomAPIURL                string
	GcomAPIToken              string
	CreateAccessPolicyTimeout time.Duration
	FetchAccessPolicyTimeout  time.Duration
	DeleteAccessPolicyTimeout time.Duration
	CreateTokenTimeout        time.Duration
}

func (cfg *Cfg) readCloudMigrationSettings() {
	cloudMigration := cfg.Raw.Section("cloud_migration")
	cfg.CloudMigration.IsTarget = cloudMigration.Key("is_target").MustBool(false)
	cfg.CloudMigration.Region = cloudMigration.Key("region").MustString("")
	cfg.CloudMigration.GcomAPIURL = cloudMigration.Key("gcom_api_url").MustString("")
	cfg.CloudMigration.GcomAPIToken = cloudMigration.Key("gcom_api_token").MustString("")
	cfg.CloudMigration.CreateAccessPolicyTimeout = cloudMigration.Key("create_access_policy_timeout").MustDuration(5 * time.Second)
	cfg.CloudMigration.FetchAccessPolicyTimeout = cloudMigration.Key("fetch_access_policy_timeout").MustDuration(5 * time.Second)
	cfg.CloudMigration.DeleteAccessPolicyTimeout = cloudMigration.Key("delete_access_policy_timeout").MustDuration(5 * time.Second)
	cfg.CloudMigration.CreateTokenTimeout = cloudMigration.Key("create_token_timeout").MustDuration(5 * time.Second)
}
