package apiv1

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func TestAHandlersProblemCauseIsLoggedOnce(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(prev)

	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		for path, p := range map[string]*problem.Problem{
			"/internal":    problem.Internal(errors.New("pq: relation topic_secret does not exist")),
			"/unavailable": problem.Unavailable(errors.New("dial tcp 10.0.0.7:443: i/o timeout")),
		} {
			huma.Register(api, Public(huma.Operation{OperationID: strings.Trim(path, "/"), Method: http.MethodGet, Path: path, Summary: "Fails"}),
				func(context.Context, *struct{}) (*struct{}, error) { return nil, p })
		}
	})
	for path, c := range map[string]struct {
		status int
		cause  string
	}{
		"/internal":    {500, "topic_secret"},
		"/unavailable": {503, "10.0.0.7"},
	} {
		logs.Reset()
		resp := do(t, app, http.MethodGet, "/api/v1"+path, "", nil)
		body := string(readBody(t, resp))
		if resp.StatusCode != c.status || strings.Contains(body, c.cause) {
			t.Errorf("%s: %d %s", path, resp.StatusCode, body)
		}
		id := resp.Header.Get(problem.HeaderRequestID)
		if n := strings.Count(logs.String(), c.cause); n != 1 || !strings.Contains(logs.String(), id) {
			t.Errorf("%s: cause logged %d times, want once under %s: %q", path, n, id, logs.String())
		}
	}
}
