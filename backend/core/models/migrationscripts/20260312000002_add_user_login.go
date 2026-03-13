/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

var _ plugin.MigrationScript = (*addUserFullName)(nil)

type addUserFullName struct{}

type user20260312000002 struct {
	Id           string `gorm:"primaryKey;type:varchar(255)"`
	UserFullName string `gorm:"type:varchar(255)"`
}

func (user20260312000002) TableName() string {
	return "users"
}

func (*addUserFullName) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&user20260312000002{},
	)
}

func (*addUserFullName) Version() uint64 {
	return 20260312000002
}

func (*addUserFullName) Name() string {
	return "add user_full_name to users table"
}
