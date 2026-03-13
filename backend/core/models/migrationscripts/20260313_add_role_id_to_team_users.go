package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

var _ plugin.MigrationScript = (*AddRoleIdToTeamUsers)(nil)

type AddRoleIdToTeamUsers struct{}

type teamUser20260313000001 struct {
	TeamId string `gorm:"primaryKey;type:varchar(255)"`
	UserId string `gorm:"primaryKey;type:varchar(255)"`
	RoleId string `gorm:"type:varchar(255)"`
}

func (teamUser20260313000001) TableName() string {
	return "team_users"
}

func (*AddRoleIdToTeamUsers) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&teamUser20260313000001{},
	)
}

func (*AddRoleIdToTeamUsers) Version() uint64 {
	return 20260313000001
}

func (*AddRoleIdToTeamUsers) Name() string {
	return "add role_id to team_users"
}
