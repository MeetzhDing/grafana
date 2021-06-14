package graphite

import (
	"encoding/json"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeRange(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"now", "now"},
		{"now-1m", "-1min"},
		{"now-1M", "-1mon"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			tr := formatTimeRange(tc.input)
			assert.Equal(t, tc.expected, tr)
		})
	}
}

func TestFixIntervalFormat(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "should transform 1m to graphite unit (1min) when used as interval string",
			target:   "aliasByNode(hitcount(averageSeries(app.grafana.*.dashboards.views.count), '1m'), 4)",
			expected: "aliasByNode(hitcount(averageSeries(app.grafana.*.dashboards.views.count), '1min'), 4)",
		},
		{
			name:     "should transform 1M to graphite unit (1mon) when used as interval string",
			target:   "aliasByNode(hitcount(averageSeries(app.grafana.*.dashboards.views.count), '1M'), 4)",
			expected: "aliasByNode(hitcount(averageSeries(app.grafana.*.dashboards.views.count), '1mon'), 4)",
		},
		{
			name:     "should not transform 1m when not used as interval string",
			target:   "app.grafana.*.dashboards.views.1m.count",
			expected: "app.grafana.*.dashboards.views.1m.count",
		},
		{
			name:     "should not transform 1M when not used as interval string",
			target:   "app.grafana.*.dashboards.views.1M.count",
			expected: "app.grafana.*.dashboards.views.1M.count",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tr := fixIntervalFormat(tc.target)
			assert.Equal(t, tc.expected, tr)
		})
	}

	executor := &GraphiteExecutor{}

	t.Run("Converts response to data frames", func(*testing.T) {
		body := `
		[
			{
				"target": "target",
				"tags": { "fooTag": "fooValue", "barTag": "barValue" },
				"datapoints": [[50, 1], [null, 2], [50, 3]]
			}
		]`
		a := 50.0
		expectedFrame := data.NewFrame("target",
			data.NewField("time", nil, []time.Time{time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC(), time.Unix(3, 0).UTC()}),
			data.NewField("value", data.Labels{
				"fooTag": "fooValue",
				"barTag": "barValue",
			}, []*float64{&a, nil, &a}).SetConfig(&data.FieldConfig{DisplayNameFromDS: "target"}),
		)
		expectedFrames := data.Frames{expectedFrame}

		httpResponse := &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(body))}
		dataFrames, err := executor.toDataFrames(httpResponse)

		require.NoError(t, err)
		if !reflect.DeepEqual(expectedFrames, dataFrames) {
			expectedFramesJSON, _ := json.Marshal(expectedFrames)
			dataFramesJSON, _ := json.Marshal(dataFrames)
			t.Errorf("Data frames should have been equal but was, expected:\n%s\nactual:\n%s", expectedFramesJSON, dataFramesJSON)
		}
	})
}
