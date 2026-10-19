package viewer

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

//go:embed templates/*.html static/*.css
var assets embed.FS

type reader interface {
	RunPage(context.Context, store.RunFilter) (store.RunPage, error)
	RunStructure(context.Context, string, int) (model.RunDetail, error)
	PhaseStart(context.Context, model.Run) (string, error)
	ViewerComparison(context.Context, string, model.BaselineOptions) (model.Comparison, error)
}
type writer interface {
	SaveDiff(context.Context, model.DiffResult) error
}
type viewer struct {
	reader    reader
	writer    writer
	templates *template.Template
	logger    *slog.Logger
	slots     chan struct{}
}

func New(read reader, write writer, logger *slog.Logger) (http.Handler, error) {
	templates, err := template.New("viewer").Funcs(template.FuncMap{"json": jsonText, "time": timestamp, "cutoff": timeValue, "runURL": runURL, "num": number}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse viewer templates: %w", err)
	}
	v := &viewer{reader: read, writer: write, templates: templates, logger: logger, slots: make(chan struct{}, 2)}
	css, err := assets.ReadFile("static/viewer.css")
	if err != nil {
		return nil, fmt.Errorf("read viewer CSS: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/runs", http.StatusSeeOther) })
	mux.HandleFunc("GET /runs", v.list)
	mux.HandleFunc("GET /run", v.run)
	mux.HandleFunc("POST /run/diff", v.compare)
	mux.HandleFunc("GET /assets/viewer.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if _, err := w.Write(css); err != nil {
			v.logger.Debug("write viewer CSS", "error", err)
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; style-src 'self'; img-src 'none'; connect-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if len(r.URL.RequestURI()) > 8192 {
			v.problem(w, http.StatusBadRequest, "URL is too long")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

func (v *viewer) render(w http.ResponseWriter, status int, name string, data any) {
	var buffer bytes.Buffer
	if err := v.templates.ExecuteTemplate(&buffer, name, data); err != nil {
		v.logger.Error("render viewer", "error", err)
		http.Error(w, "Unable to render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		v.logger.Debug("write viewer page", "error", err)
	}
}
func (v *viewer) problem(w http.ResponseWriter, status int, message string) {
	v.render(w, status, "error.html", struct{ Title, Message string }{Title: http.StatusText(status), Message: message})
}
func (v *viewer) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		v.problem(w, 404, "Run or evidence is no longer available.")
	case errors.Is(err, store.EvidenceLimit):
		v.problem(w, 422, "This run or comparison exceeds the viewer evidence limit. Full evidence remains available through the CLI.")
	case errors.Is(err, store.QueueFull):
		w.Header().Set("Retry-After", "1")
		v.problem(w, 503, "The writer queue is busy. Try again shortly.")
	case errors.Is(err, store.RunChanged):
		v.problem(w, 409, "The failing run changed during comparison. Reload the run before trying again.")
	case errors.Is(err, context.DeadlineExceeded):
		v.problem(w, 504, "The viewer request exceeded its time limit.")
	default:
		v.logger.Error("viewer request", "error", err)
		v.problem(w, 500, "Unable to load this view.")
	}
}

func parseTime(value string) (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0, fmt.Errorf("time must be RFC3339")
	}
	ns := t.UnixNano()
	if !time.Unix(0, ns).Equal(t) {
		return 0, fmt.Errorf("time outside nanosecond range")
	}
	return ns, nil
}

func listFilter(query url.Values) (store.RunFilter, error) {
	now := time.Now().UTC()
	f := store.RunFilter{Workload: query.Get("workload"), Outcome: query.Get("outcome"), From: now.Add(-7 * 24 * time.Hour).UnixNano(), To: now.UnixNano(), Limit: 25}
	if len(f.Workload) > 512 {
		return f, fmt.Errorf("workload is too long")
	}
	var err error
	if text := query.Get("to"); text != "" {
		f.To, err = parseTime(text)
		if err != nil {
			return f, err
		}
		f.From = time.Unix(0, f.To).Add(-7 * 24 * time.Hour).UnixNano()
	}
	if text := query.Get("from"); text != "" {
		f.From, err = parseTime(text)
		if err != nil {
			return f, err
		}
	}
	if text := query.Get("limit"); text != "" {
		f.Limit, err = strconv.Atoi(text)
		if err != nil {
			return f, fmt.Errorf("invalid page size")
		}
	}
	if f.From >= f.To || time.Unix(0, f.To).Sub(time.Unix(0, f.From)) > 31*24*time.Hour || f.Limit < 1 || f.Limit > 100 {
		return f, fmt.Errorf("choose 1..100 rows and a time range of at most 31 days")
	}
	if f.Outcome != "" && f.Outcome != "pass" && f.Outcome != "fail" && f.Outcome != "degraded" {
		return f, fmt.Errorf("unknown outcome")
	}
	if text := query.Get("cursor"); text != "" {
		if len(text) > 4096 {
			return f, fmt.Errorf("invalid cursor")
		}
		data, err := base64.RawURLEncoding.DecodeString(text)
		if err != nil {
			return f, fmt.Errorf("invalid cursor")
		}
		var cursor store.RunCursor
		if err = json.Unmarshal(data, &cursor); err != nil || cursor.ID == "" || cursor.StartedAt < f.From || cursor.StartedAt >= f.To {
			return f, fmt.Errorf("invalid cursor")
		}
		f.After = &cursor
	}
	return f, nil
}

func (v *viewer) list(w http.ResponseWriter, r *http.Request) {
	filter, err := listFilter(r.URL.Query())
	if err != nil {
		v.problem(w, 400, err.Error())
		return
	}
	page, err := v.reader.RunPage(r.Context(), filter)
	if err != nil {
		v.failure(w, err)
		return
	}
	view := listView{Title: "Runs", Workload: filter.Workload, Outcome: filter.Outcome, From: timeValue(filter.From), To: timeValue(filter.To), Limit: filter.Limit}
	for _, run := range page.Runs {
		view.Rows = append(view.Rows, runRow{Run: run, URL: runURL(run.ID)})
	}
	query := url.Values{"workload": {filter.Workload}, "outcome": {filter.Outcome}, "from": {view.From}, "to": {view.To}, "limit": {strconv.Itoa(filter.Limit)}}
	view.First = "/runs?" + query.Encode()
	if page.Next != nil {
		data, err := json.Marshal(page.Next)
		if err != nil {
			v.failure(w, err)
			return
		}
		query.Set("cursor", base64.RawURLEncoding.EncodeToString(data))
		view.Next = "/runs?" + query.Encode()
	}
	v.render(w, 200, "runs.html", view)
}

func (v *viewer) loadRun(ctx context.Context, id string) (runView, error) {
	detail, err := v.reader.RunStructure(ctx, id, store.ViewerSignalLimit)
	if err != nil {
		return runView{}, err
	}
	before, err := v.reader.PhaseStart(ctx, detail.Run)
	if err != nil {
		return runView{}, err
	}
	spans, orphans := timeline(detail)
	return runView{Title: "Run " + id, DiffAction: "/run/diff?id=" + url.QueryEscape(id) + "#diff", Before: before, Run: detail.Run, Spans: spans, Orphans: orphans}, nil
}

func (v *viewer) run(w http.ResponseWriter, r *http.Request) {
	view, err := v.loadRun(r.Context(), r.URL.Query().Get("id"))
	if err != nil {
		v.failure(w, err)
		return
	}
	v.render(w, 200, "run.html", view)
}

func sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return err == nil && u.Scheme == scheme && strings.EqualFold(u.Host, r.Host) && u.User == nil
}

func (v *viewer) compare(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		v.problem(w, 403, "Diff computation requires the same origin.")
		return
	}
	// Limit CPU work independently of the intake queue. A full slot channel
	// rejects extra requests; no unbounded background computations are spawned.
	select {
	case v.slots <- struct{}{}:
		defer func() { <-v.slots }()
	default:
		w.Header().Set("Retry-After", "1")
		v.problem(w, 429, "Two comparisons are already running. Try again shortly.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		v.problem(w, 400, "Invalid comparison form")
		return
	}
	view, err := v.loadRun(r.Context(), r.URL.Query().Get("id"))
	if err != nil {
		v.failure(w, err)
		return
	}
	if !view.Run.Closed || view.Run.Outcome != "fail" {
		v.problem(w, 409, "Comparison requires a closed failing run.")
		return
	}
	if len(r.PostForm.Get("baseline_before")) > 2048 {
		v.problem(w, 400, "Cutoff is too long")
		return
	}
	if before := r.PostForm.Get("baseline_before"); before != "" {
		view.Before = before
	}
	for _, mode := range []string{"before", "default"} {
		title := "Before the failure phase"
		options := model.BaselineOptions{}
		if mode == "before" {
			if view.Before == "" {
				view.Panels = append(view.Panels, panel{Title: title, Message: "Supply a cutoff run ID or RFC3339 time to compare before the failure phase."})
				continue
			}
			options.Before = view.Before
		} else {
			title = "Recent baseline"
		}
		input, err := v.reader.ViewerComparison(r.Context(), view.Run.ID, options)
		if err != nil {
			if mode == "before" && (errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.InvalidCutoff)) {
				v.problem(w, 400, "Cutoff must be an existing run ID or RFC3339 time.")
				return
			}
			v.failure(w, err)
			return
		}
		result, err := diff.Compute(input)
		if err != nil {
			v.failure(w, err)
			return
		}
		if err = v.writer.SaveDiff(r.Context(), result); err != nil {
			v.failure(w, err)
			return
		}
		display, err := panelView(title, input, result)
		if err != nil {
			v.failure(w, err)
			return
		}
		view.Panels = append(view.Panels, display)
	}
	v.render(w, 200, "run.html", view)
}
