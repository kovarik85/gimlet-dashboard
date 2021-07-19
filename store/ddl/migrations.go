// Copyright 2019 Laszlo Fogas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ddl

const createTableUsers = "create-table-users"
const createTableComponentStates = "create-table-componentStates"
const addPrefsColumnToUsersTable = "add-prefs-column-to-users-table"
const addComponentStateMapToComponentStateTable = "add-component-state-map-to-component-state-table"
const addWrittenGitopsReposToUsersTable = "add-written-gitops-repos-to-users-table"
const addInstallationsToUsersTable = "add-installations-to-users-table"
const dropComponentStateFromComponentStateTable = "drop-component-state-from-component-state-table"
const dropGitComponentStateFromComponentStateTable = "drop-git-component-state-from-component-state-table"

type migration struct {
	name string
	stmt string
}

var migrations = map[string][]migration{
	"sqlite3": {
		{
			name: createTableUsers,
			stmt: `
CREATE TABLE IF NOT EXISTS users (
id           INTEGER PRIMARY KEY AUTOINCREMENT,
login         TEXT,
email         TEXT,
access_token  TEXT,
refresh_token TEXT,
expires       INT,
secret        TEXT,
UNIQUE(login)
);
`,
		},
	},
	"postgres": {},
}