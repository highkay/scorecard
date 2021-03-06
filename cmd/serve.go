// Copyright 2020 Security Scorecard Authors
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

package cmd

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ossf/scorecard/checks"
	"github.com/ossf/scorecard/pkg"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the scorecard program over http",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := zap.NewProductionConfig()
		cfg.Level.SetLevel(*logLevel)
		logger, _ := cfg.Build()
		//nolint
		defer logger.Sync() // flushes buffer, if any
		sugar := logger.Sugar()
		t, err := template.New("webpage").Parse(tpl)
		if err != nil {
			sugar.Panic(err)
		}

		http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
			repoParam := r.URL.Query().Get("repo")
			s := strings.SplitN(repoParam, "/", 3)
			if len(s) != 3 {
				rw.WriteHeader(http.StatusBadRequest)
			}
			sugar.Info(repoParam)
			repo := pkg.RepoURL{
				Host:  s[0],
				Owner: s[1],
				Repo:  s[2],
			}
			ctx := r.Context()
			resultCh := pkg.RunScorecards(ctx, sugar, repo, checks.AllChecks)
			tc := tc{
				URL: repoParam,
			}
			for r := range resultCh {
				sugar.Info(r)
				tc.Results = append(tc.Results, r)
			}
			if err := t.Execute(rw, tc); err != nil {
				sugar.Warn(err)
			}
		})
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		fmt.Printf("Listening on localhost:%s\n", port)
		err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil)
		if err != nil {
			log.Fatal("ListenAndServe ", err)
		}
	},
}

type tc struct {
	URL     string
	Results []pkg.Result
}

const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Scorecard Results for: {{.URL}}</title>
	</head>
	<body>
		{{range .Results}}
			<div>
				<p>{{ .Name }}: {{ .Cr.Pass }}</p>
			</div>
		{{end}}
	</body>
</html>`
