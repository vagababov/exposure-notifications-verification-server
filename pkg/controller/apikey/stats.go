// Copyright 2020 Google LLC
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

package apikey

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/exposure-notifications-verification-server/internal/project"
	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
	"github.com/google/exposure-notifications-verification-server/pkg/rbac"
	"github.com/gorilla/mux"
)

func (c *Controller) HandleStats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)

		session := controller.SessionFromContext(ctx)
		if session == nil {
			controller.MissingSession(w, r, c.h)
			return
		}

		membership := controller.MembershipFromContext(ctx)
		if membership == nil {
			controller.MissingMembership(w, r, c.h)
			return
		}
		if !(membership.Can(rbac.APIKeyRead) || membership.Can(rbac.StatsRead)) {
			controller.Unauthorized(w, r, c.h)
			return
		}
		currentRealm := membership.Realm

		// Pull the authorized app from the id.
		authApp, err := currentRealm.FindAuthorizedApp(c.db, vars["id"])
		if err != nil {
			if database.IsNotFound(err) {
				controller.Unauthorized(w, r, c.h)
				return
			}

			controller.InternalError(w, r, c.h, err)
			return
		}

		stats, err := authApp.StatsCached(ctx, c.db, c.cacher)
		if err != nil {
			controller.InternalError(w, r, c.h, err)
			return
		}

		pth := r.URL.Path
		switch {
		case strings.HasSuffix(pth, ".csv"):
			nowFormatted := time.Now().UTC().Format(project.RFC3339Squish)
			filename := fmt.Sprintf("%s-apikey-stats.csv", nowFormatted)
			c.h.RenderCSV(w, http.StatusOK, filename, stats)
			return
		case strings.HasSuffix(pth, ".json"):
			c.h.RenderJSON(w, http.StatusOK, stats)
			return
		default:
			controller.NotFound(w, r, c.h)
			return
		}
	})
}
