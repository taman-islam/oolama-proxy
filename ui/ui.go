package ui

import (
	"html/template"
	"lb/auth"
	"lb/limiter"
	"lb/store"
	"net/http"

	"github.com/labstack/echo/v4"
)

const rawDashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Proxy Admin Dashboard</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    background: #0f0f13; color: #e2e8f0; min-height: 100vh; padding: 2rem;
  }
  h1 { font-size: 1.5rem; font-weight: 700; color: #7c3aed; margin-bottom: 0.25rem; }
  .subtitle { color: #64748b; font-size: 0.875rem; margin-bottom: 2rem; }
  .card {
    background: #1a1a24; border: 1px solid #2d2d3d;
    border-radius: 12px; padding: 1.5rem; margin-bottom: 2rem;
  }
  .card h2 { font-size: 1rem; font-weight: 600; color: #a78bfa; margin-bottom: 1rem; }
  table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
  th { text-align: left; padding: 0.5rem 0.75rem; color: #64748b; font-weight: 500; border-bottom: 1px solid #2d2d3d; }
  td { padding: 0.6rem 0.75rem; border-bottom: 1px solid #1e1e2e; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #1e1e2e; }
  .tag { display: inline-block; padding: 0.15rem 0.55rem; border-radius: 999px; font-size: 0.75rem; font-weight: 500; }
  .tag-purple { background: #3b0764; color: #c4b5fd; }
  .quota-bar-wrap { background: #0f0f13; border-radius: 999px; height: 6px; width: 120px; display: inline-block; vertical-align: middle; margin-left: 0.5rem; }
  .quota-bar { height: 6px; border-radius: 999px; background: #7c3aed; }
  .inf { color: #64748b; font-style: italic; }
</style>
</head>
<body>
<h1>🔮 Proxy Admin Dashboard</h1>
<p class="subtitle">Ollama OpenAI-Compatible Proxy &mdash; live view</p>

<div class="card">
  <h2>Usage by User &amp; Model</h2>
  <table>
    <thead><tr><th>User</th><th>Model</th><th>Prompt Tokens</th><th>Completion Tokens</th><th>Total</th></tr></thead>
    <tbody>
    {{- range $user, $models := .Usage}}
      {{- range $model, $u := $models}}
      <tr>
        <td><span class="tag tag-purple">{{$user}}</span></td>
        <td>{{$model}}</td>
        <td>{{$u.PromptTokens}}</td>
        <td>{{$u.CompletionTokens}}</td>
        <td>{{add $u.PromptTokens $u.CompletionTokens}}</td>
      </tr>
      {{- end}}
    {{- else}}
      <tr><td colspan="5" style="color:#64748b;text-align:center;padding:1.5rem">No usage recorded yet.</td></tr>
    {{- end}}
    </tbody>
  </table>
</div>

<div class="card">
  <h2>Rate &amp; Quota Limits</h2>
  <table>
    <thead><tr><th>User</th><th>RPS Limit</th><th>Token Quota</th><th>Tokens Used</th><th>Remaining</th></tr></thead>
    <tbody>
    {{- range $user, $info := .Limits}}
      <tr>
        <td><span class="tag tag-purple">{{$user}}</span></td>
        <td>{{if eq $info.RPS 0.0}}<span class="inf">∞</span>{{else}}{{printf "%.0f" $info.RPS}}/s{{end}}</td>
        <td>{{if eq $info.MaxTokens 0}}<span class="inf">∞</span>{{else}}{{$info.MaxTokens}}{{end}}</td>
        <td>{{$info.UsedTokens}}</td>
        <td>
          {{- if eq $info.MaxTokens 0}}<span class="inf">∞</span>
          {{- else}}
            {{remaining $info.MaxTokens $info.UsedTokens}}
            <span class="quota-bar-wrap"><div class="quota-bar" style="width:{{pct $info.UsedTokens $info.MaxTokens}}%"></div></span>
          {{- end}}
        </td>
      </tr>
    {{- else}}
      <tr><td colspan="5" style="color:#64748b;text-align:center;padding:1.5rem">No limits configured.</td></tr>
    {{- end}}
    </tbody>
  </table>
</div>
<p style="color:#334155;font-size:0.75rem">Reload page to refresh &bull; All data is in-memory only</p>
</body>
</html>`

type dashboardData struct {
	Usage  map[string]map[string]store.ModelUsage
	Limits map[string]limiter.LimitInfo
}

// Dashboard handles GET /admin/ui — renders a live usage + limits overview.
func Dashboard(s *store.Store, lim *limiter.Limiter) echo.HandlerFunc {
	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"remaining": func(max, used int64) int64 {
			if r := max - used; r > 0 {
				return r
			}
			return 0
		},
		"pct": func(used, max int64) int64 {
			if max == 0 {
				return 0
			}
			if p := used * 100 / max; p <= 100 {
				return p
			}
			return 100
		},
	}
	tmpl := template.Must(template.New("dashboard").Funcs(funcs).Parse(rawDashboardTemplate))

	return func(c echo.Context) error {
		// cheaply ensure viewer is admin
		if !c.Get(auth.AdminCtxKey).(bool) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin access required"})
		}
		data := dashboardData{
			Usage:  s.GetAll(),
			Limits: lim.GetAllLimits(),
		}
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		return tmpl.Execute(c.Response().Writer, data)
	}
}
