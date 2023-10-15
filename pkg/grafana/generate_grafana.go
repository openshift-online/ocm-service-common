package generate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/getkin/kin-openapi/openapi3"
)

// toPtr returns a pointer copy of value.
func toPtr[T any](v T) *T {
	return &v
}

func GenerateGrafana(configFile string, outputFile string) {
	var (
		config                 *Config
		doc                    *openapi3.T
		rowNum, panelId, fldId int
		err                    error
		panel                  Panel
		panelItem              ConfigPanelItem
	)

	config, err = loadConfig(configFile)
	logErrorAndExit(err)

	var grafana = Grafana{
		Id:                   toPtr(20883),
		Uid:                  config.Grafana.Uid,
		Title:                config.Grafana.Title,
		Style:                config.Grafana.Style,
		SchemaVersion:        config.Grafana.SchemaVersion,
		Version:              config.Grafana.Version,
		Timezone:             config.Grafana.Timezone,
		Editable:             toPtr(true),
		GraphTooltip:         toPtr(0),
		FiscalYearStartMonth: toPtr(0),
		Iteration:            toPtr(time.Now().UnixMilli()),
		LiveNow:              toPtr(false),
		Time:                 config.Time,
		Timepicker:           config.Timepicker,
		Templating:           config.Templating,
		Annotations:          config.Annotations,
		Requires:             config.Requires,
	}

	fldId = 1
	rowNum = -1
	panelId = 0
	datasource := config.PanelDatasource
	for _, panelItem = range *config.Panels {
		switch *panelItem.Type {
		case "5items":
			rowNum++
			panelId++
			updateGrafanaService(&panelItem, config.Grafana.Service)
			fldId, panel = createRegular(*datasource, *panelItem.Path, rowNum, panelId, fldId,
				*panelItem.Availability,
				*panelItem.Requests,
				*panelItem.Errors1, *panelItem.Errors2, panelItem.Errors3,
				*panelItem.Duration,
				*panelItem.Latency,
			)
			grafana.Panels = append(grafana.Panels, panel)
		case "openapi":
			// Load and validate openapi
			loader := openapi3.NewLoader()
			loader.IsExternalRefsAllowed = true
			doc, err = loader.LoadFromFile(*panelItem.Path)
			logErrorAndExit(err)
			err = doc.Validate(loader.Context)
			logErrorAndExit(err)
			paths := keys(doc.Paths)
			sort.StringSlice(paths).Sort()
			skipSA := false
			if config.Grafana.SkipServiceAccount != nil {
				skipSA = *config.Grafana.SkipServiceAccount
			}
			for _, path := range paths {
				exprAvailability, exprRequests,
					exprErrors1, exprErrors2, exprError3,
					exprDuration, exprLatency := prepareExpressions(config.Grafana.Service, skipSA, normalizePath(path), *panelItem.Exception)
				rowNum++
				panelId++
				fldId, panel = createRegular(*datasource, path, rowNum, panelId, fldId,
					exprAvailability,
					exprRequests,
					exprErrors1, exprErrors2, exprError3,
					exprDuration,
					exprLatency,
				)
				grafana.Panels = append(grafana.Panels, panel)
			}

		case "custom":
			fldId, panel = createCustom(*panelItem.Path, *datasource, *panelItem.Panels, rowNum, panelId, fldId, config.Grafana.Service)
			rowNum++
			panelId++
			grafana.Panels = append(grafana.Panels, panel)
		}
	}

	printGrafana(config, grafana, outputFile)
}

func createCustom(title string, datasource Datasource, panels []PanelItem, rowNum int, panelId int, fldId int, service string) (int, Panel) {
	panel := Panel{
		Title:      title,
		Type:       "row",
		Collapsed:  true,
		Datasource: datasource,
		GridPos: GridPos{
			H: 1,
			W: 24,
			X: 0,
			Y: rowNum,
		},
		Id:     panelId,
		Panels: []PanelItem{},
	}
	for _, pi := range panels {
		pi.Id = fldId
		pi.PluginVersion = toPtr("7.2.1")
		refId := 'A'
		for i := range pi.Targets {
			pi.Targets[i].Expr = strings.Replace(pi.Targets[i].Expr, "grafana.service", service, -1)
			pi.Targets[i].RefId = fmt.Sprintf("%c", refId)
			refId++
		}
		panel.Panels = append(panel.Panels, pi)
		fldId++
	}

	return fldId, panel
}

func createRegular(datasource Datasource, title string, rowNum int, panelId int, fldId int,
	exprAvailability string, exprRequests string,
	exprErrors1 string, exprErrors2 string, exprErrors3 *string,
	exprDuration string, exprLatency string) (int, Panel) {

	panel := Panel{
		Type:  "row",
		Title: title,
		GridPos: GridPos{
			H: 1,
			W: 24,
			X: 0,
			Y: rowNum,
		},
		Id:        panelId,
		Collapsed: true,
		Datasource: Datasource{
			Type: datasource.Type,
			Uid:  datasource.Uid,
		},
		Panels: []PanelItem{
			{
				Datasource: &Datasource{
					Uid: toPtr("$datasource"),
				},
				Description: toPtr("The percentage of time that the API is available (not returning 500 Internal Service Errors)."),
				GridPos: GridPos{
					H: 4,
					W: 3,
					X: 0,
					Y: 1,
				},
				FieldConfig: &FieldConfig{
					Defaults: &Defaults{
						Color: &Color{
							Mode: toPtr("thresholds"),
						},
						Mappings: &[]Mapping{
							{
								Options: &MappingOptions{
									Match: toPtr("null"),
									Result: &Result{
										Text: toPtr("N/A"),
									},
								},
								Type: toPtr("special"),
							},
						},
						Thresholds: &Threshold{
							Mode: toPtr("absolute"),
							Steps: &[]Step{
								{
									Color: toPtr("#d44a3a"),
									Value: toPtr(float32(0)),
								},
								{
									Color: toPtr("rgba(237, 129, 40, 0.89)"),
									Value: toPtr(float32(0.99)),
								},
								{
									Color: toPtr("#299c46"),
									Value: toPtr(float32(0.995)),
								},
							},
						},
						Unit:      toPtr("percentunit"),
						Overrides: nil,
					},
					Overrides: &[]string{},
				},
				Id:            fldId,
				MaxDataPoints: toPtr(100),
				Options: &Options{
					ColorMode:   toPtr("value"),
					GraphMode:   toPtr("none"),
					JustifyMode: toPtr("auto"),
					Orientation: toPtr("horizontal"),
					ReduceOptions: &ReduceOptions{
						Calcs: &[]string{"lastNotNull"},
					},
					TextMode: toPtr("auto"),
				},
				PluginVersion: toPtr("8.5.2"),
				Targets: []Target{
					{
						Expr:    exprAvailability,
						Format:  toPtr("time_series"),
						Instant: toPtr(true),
						RefId:   "A",
					},
				},
				Title: toPtr("Availability (selected time) "),
				Type:  toPtr("stat"),
			},
			{
				Bars:        toPtr(false),
				DashLengths: toPtr(10),
				Dashes:      toPtr(false),
				Datasource: &Datasource{
					Uid: toPtr("$datasource"),
				},
				Description:  toPtr("Number of requests per second by response codes."),
				Fill:         toPtr(1),
				FillGradient: toPtr(0),
				GridPos: GridPos{
					H: 8,
					W: 7,
					X: 3,
					Y: 1,
				},
				HiddenSeries: toPtr(false),
				Id:           fldId + 1,
				Legend: &Legend{
					Avg:     toPtr(false),
					Current: toPtr(false),
					Max:     toPtr(false),
					Min:     toPtr(false),
					Show:    toPtr(true),
					Total:   toPtr(false),
					Values:  toPtr(false),
				},
				Lines:         toPtr(true),
				LineWidth:     toPtr(1),
				NullPointMode: toPtr("null as zero"),
				Options: &Options{
					AlertThreshold: toPtr(true),
				},
				Percentage:    toPtr(false),
				PluginVersion: toPtr("8.5.2"),
				PointRadius:   toPtr(5),
				Points:        toPtr(false),
				Renderer:      toPtr("flot"),
				SpaceLength:   toPtr(10),
				Stack:         toPtr(false),
				SteppedLine:   toPtr(false),
				Targets: []Target{
					{
						Expr:         exprRequests,
						LegendFormat: toPtr("{{code}} - {{method}} - {{deployment_ring}}"),
						RefId:        "A",
					},
				},
				Title: toPtr("Requests"),
				Tooltip: &Tooltip{
					Shared:    toPtr(true),
					Sort:      toPtr(0),
					ValueType: toPtr("individual"),
				},
				Type: toPtr("graph"),
				XAxis: &XAxis{
					Mode:   toPtr("time"),
					Show:   toPtr(true),
					Values: &[]string{},
				},
				Yaxes: &[]YAxe{
					{
						Format:  toPtr("reqps"),
						LogBase: toPtr(1),
						Show:    toPtr(true),
					},
					{
						Format:  toPtr("short"),
						LogBase: toPtr(1),
						Show:    toPtr(true),
					},
				},
				YAxis: &YAxis{
					Align: toPtr(false),
				},
			},
			{
				Bars:        toPtr(false),
				DashLengths: toPtr(10),
				Dashes:      toPtr(false),
				Datasource: &Datasource{
					Uid: toPtr("$datasource"),
				},
				Description:  toPtr("The error percentage."),
				Fill:         toPtr(0),
				FillGradient: toPtr(0),
				GridPos: GridPos{
					H: 8,
					W: 7,
					X: 10,
					Y: 1,
				},
				HiddenSeries: toPtr(false),
				Id:           fldId + 2,
				Legend: &Legend{
					Avg:     toPtr(false),
					Current: toPtr(false),
					Max:     toPtr(false),
					Min:     toPtr(false),
					Show:    toPtr(true),
					Total:   toPtr(false),
					Values:  toPtr(false),
				},
				Lines:         toPtr(true),
				LineWidth:     toPtr(1),
				NullPointMode: toPtr("null as zero"),
				Options: &Options{
					AlertThreshold: toPtr(true),
				},
				Percentage:    toPtr(false),
				PluginVersion: toPtr("8.5.2"),
				PointRadius:   toPtr(5),
				Points:        toPtr(false),
				Renderer:      toPtr("flot"),
				SpaceLength:   toPtr(10),
				Stack:         toPtr(false),
				SteppedLine:   toPtr(false),
				Targets: []Target{
					{
						Expr:         exprErrors1,
						LegendFormat: toPtr("non-2xx - {{deployment_ring}}"),
						RefId:        "C",
					},
					{
						Expr:         exprErrors2,
						LegendFormat: toPtr("non-2xx and non-404 - {{deployment_ring}}"),
						RefId:        "A",
					},
					// Here can be exprErrors3
				},
				Title: toPtr("Errors"),
				Tooltip: &Tooltip{
					Shared:    toPtr(true),
					Sort:      toPtr(0),
					ValueType: toPtr("individual"),
				},
				Type: toPtr("graph"),
				XAxis: &XAxis{
					Mode:   toPtr("time"),
					Show:   toPtr(true),
					Values: &[]string{},
				},
				Yaxes: &[]YAxe{
					{
						Format:  toPtr("percentunit"),
						LogBase: toPtr(1),
						Min:     toPtr("0"),
						Show:    toPtr(true),
					},
					{
						Format:  toPtr("short"),
						LogBase: toPtr(1),
						Show:    toPtr(true),
					},
				},
				YAxis: &YAxis{
					Align: toPtr(false),
				},
			},
			{
				Bars:        toPtr(false),
				DashLengths: toPtr(10),
				Dashes:      toPtr(false),
				Datasource: &Datasource{
					Uid: toPtr("$datasource"),
				},
				Description: toPtr("The request duration within which the API have served 99%, 95%, 50% of requests"),
				GridPos: GridPos{
					H: 8,
					W: 7,
					X: 17,
					Y: 1,
				},
				HiddenSeries: toPtr(false),
				Id:           fldId + 3,
				Legend: &Legend{
					Avg:     toPtr(false),
					Current: toPtr(false),
					Max:     toPtr(false),
					Min:     toPtr(false),
					Show:    toPtr(true),
					Total:   toPtr(false),
					Values:  toPtr(false),
				},
				Lines:         toPtr(true),
				LineWidth:     toPtr(1),
				NullPointMode: toPtr("null as zero"),
				Options: &Options{
					AlertThreshold: toPtr(true),
				},
				Percentage:    toPtr(false),
				PluginVersion: toPtr("8.5.1"),
				PointRadius:   toPtr(5),
				Points:        toPtr(false),
				Renderer:      toPtr("flot"),
				SpaceLength:   toPtr(10),
				Stack:         toPtr(false),
				SteppedLine:   toPtr(false),
				Targets: []Target{
					{
						Expr:         exprDuration,
						Interval:     toPtr(""),
						LegendFormat: toPtr("Avg. response duration (1m) - {{deployment_ring}}"),
						RefId:        "A",
					},
				},
				Title: toPtr("Duration"),
				Tooltip: &Tooltip{
					Shared:    toPtr(true),
					Sort:      toPtr(0),
					ValueType: toPtr("individual"),
				},
				Type: toPtr("graph"),
				XAxis: &XAxis{
					Mode:   toPtr("time"),
					Show:   toPtr(true),
					Values: toPtr([]string{}),
				},
				Yaxes: &[]YAxe{
					{
						Format:  toPtr("s"),
						LogBase: toPtr(10),
						Show:    toPtr(true),
					},
					{
						Format:  toPtr("short"),
						LogBase: toPtr(1),
						Show:    toPtr(true),
					},
				},
				YAxis: &YAxis{
					Align: toPtr(false),
				},
			},
			{
				Datasource: &Datasource{
					Uid: toPtr("$datasource"),
				},
				Description: toPtr("The percentage of time that the API responds within 1 second."),
				FieldConfig: &FieldConfig{
					Defaults: &Defaults{
						Color: &Color{
							Mode: toPtr("thresholds"),
						},
						Mappings: &[]Mapping{
							{
								Options: &MappingOptions{
									Match: toPtr("null"),
									Result: &Result{
										Text: toPtr("N/A"),
									},
								},
								Type: toPtr("special"),
							},
						},
						Thresholds: &Threshold{
							Mode: toPtr("absolute"),
							Steps: &[]Step{
								{
									Color: toPtr("#d44a3a"),
								},
								{
									Color: toPtr("rgba(237, 129, 40, 0.89)"),
									Value: toPtr(float32(0.99)),
								},
								{
									Color: toPtr("#299c46"),
									Value: toPtr(float32(0.995)),
								},
							},
						},
						Unit: toPtr("percentunit"),
					},
				},
				GridPos: GridPos{
					H: 4,
					W: 3,
					X: 0,
					Y: 5,
				},
				Id:            fldId + 4,
				MaxDataPoints: toPtr(100),
				Options: &Options{
					ColorMode:   toPtr("value"),
					GraphMode:   toPtr("none"),
					JustifyMode: toPtr("auto"),
					Orientation: toPtr("horizontal"),
					ReduceOptions: &ReduceOptions{
						Calcs:  &[]string{"lastNotNull"},
						Fields: toPtr(""),
						Values: toPtr(false),
					},
					TextMode: toPtr("auto"),
				},
				PluginVersion: toPtr("8.5.2"),
				Targets: []Target{
					{
						Expr:    exprLatency,
						Format:  toPtr("time_series"),
						Instant: toPtr(true),
						RefId:   "A",
					},
				},
				Title: toPtr("Latency (< 1s) "),
				Type:  toPtr("stat"),
			},
		},
	}

	if exprErrors3 != nil {
		var legendFormat = "5xx and timeout - {{deployment_ring}}"
		if strings.Contains(*exprErrors3, "4..|0") {
			legendFormat = "4xx - {{deployment_ring}}"
		}
		panel.Panels[2].Targets = append(panel.Panels[2].Targets, Target{
			Expr:         *exprErrors3,
			LegendFormat: &legendFormat,
			RefId:        "B",
		})
	}

	return fldId + 5, panel
}

func printGrafana(config *Config, grafana Grafana, output string) {
	b, _ := json.Marshal(grafana)
	var w io.Writer
	if output != "" {
		w, _ = os.Create(output)
	} else {
		w = os.Stdout
	}
	_, _ = fmt.Fprintf(w, `apiVersion: %s
data:
  %s.json: |-
    `, config.Grafana.ApiVersion, config.Grafana.Metadata.Name)
	_, _ = w.Write(b)
	_, _ = fmt.Fprintf(w, `
kind: ConfigMap
metadata:
  name: %s
  labels:
    grafana_dashboard: "%s"
  annotations:
    grafana-folder: %s
`, config.Grafana.Metadata.Name,
		config.Grafana.Metadata.Labels.GrafanaDashboard,
		config.Grafana.Metadata.Annotations.GrafanaFolder,
	)
}

func logErrorAndExit(err error, values ...interface{}) {
	if err != nil {
		fmt.Printf(err.Error()+"\n", values...)
		os.Exit(1)
	}
}

// MarshalJSON used due to Query having two JSON forms
func (q *Query) MarshalJSON() ([]byte, error) {
	if q.RefId == nil {
		return json.Marshal(q.Query)
	} else {
		return json.Marshal(&struct {
			Query string `json:"query,omitempty"`
			RefId string `json:"refId,omitempty"`
		}{
			Query: *q.Query,
			RefId: *q.RefId,
		})
	}
}

var reNormalizePath = regexp.MustCompile(`\{(.*?)}`)

func normalizePath(path string) string {
	return reNormalizePath.ReplaceAllString(path, "-")
}

func prepareExpressions(service string, skipServiceAccount bool, path string, exception Exception) (string, string, string, string, *string, string, string) {
	exprAvailability := fmt.Sprintf(`
sum(increase(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s",code!~"5..|0"SA}[$__range]))
/
sum(increase(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s"SA}[$__range]))
`,
		service, path, service, path)

	exprRequests := fmt.Sprintf(`
sum by (code, method) (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s"SA}[$__range]))
`,
		service, path)

	exprErrors1 := fmt.Sprintf(`
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s",code!~"2.."SA}[$__range]))
/
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s"SA}[$__range]))
`,
		service, path, service, path)

	exprErrors2 := fmt.Sprintf(`
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s",code=~"5..|0",service_account=~"$account"}[$__range]))
/
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s",service_account=~"$account"}[$__range]))
`,
		service, path, service, path)

	exprDuration := fmt.Sprintf(`
sum (increase(api_inbound_request_duration_sum{namespace="$namespace",service=~"%s",path="%s",code!~"5..|0"SA}[$__range]))
/
sum (increase(api_inbound_request_duration_count{namespace="$namespace",service=~"%s",path="%s",code!~"5..|0"SA}[$__range]))
`,
		service, path, service, path)

	exprLatency := fmt.Sprintf(`
sum(increase(api_inbound_request_duration_bucket{namespace="$namespace",service=~"%s",path="%s",code!~"5..|0",le="1"SA}[$__range]))
/
sum(increase(api_inbound_request_duration_count{namespace="$namespace",service=~"%s",path="%s",code!~"5..|0"SA}[$__range]))
`,
		service, path, service, path)

	var exprError3 *string
	if strings.Contains(path, exception.Path) {
		exception.Errors3 = toPtr(strings.Replace(*exception.Errors3, "grafana.service", service, -1))
		exprError3 = toPtr(fmt.Sprintf(*exception.Errors3, path, exception.Method, path, exception.Method))
	} else {
		exprError3 = toPtr(fmt.Sprintf(`
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s",code=~"4..|0"SA}[$__range]))
/
sum (rate(api_inbound_request_count{namespace="$namespace",service=~"%s",path="%s"SA}[$__range]))
`, service, path, service, path))
	}

	if skipServiceAccount {
		exprAvailability = strings.ReplaceAll(exprAvailability, "SA", "")
		exprRequests = strings.ReplaceAll(exprRequests, "SA", "")
		exprErrors1 = strings.ReplaceAll(exprErrors1, "SA", "")
		exprErrors2 = strings.ReplaceAll(exprErrors2, "SA", "")
		*exprError3 = strings.ReplaceAll(*exprError3, "SA", "")
		exprDuration = strings.ReplaceAll(exprDuration, "SA", "")
		exprLatency = strings.ReplaceAll(exprLatency, "SA", "")
	} else {
		exprAvailability = strings.ReplaceAll(exprAvailability, "SA", ",service_account=~\"$account\"")
		exprRequests = strings.ReplaceAll(exprRequests, "SA", ",service_account=~\"$account\"")
		exprErrors1 = strings.ReplaceAll(exprErrors1, "SA", ",service_account=~\"$account\"")
		exprErrors2 = strings.ReplaceAll(exprErrors2, "SA", ",service_account=~\"$account\"")
		*exprError3 = strings.ReplaceAll(*exprError3, "SA", ",service_account=~\"$account\"")
		exprDuration = strings.ReplaceAll(exprDuration, "SA", ",service_account=~\"$account\"")
		exprLatency = strings.ReplaceAll(exprLatency, "SA", ",service_account=~\"$account\"")
	}

	return exprAvailability, exprRequests, exprErrors1, exprErrors2, exprError3, exprDuration, exprLatency
}

func loadConfig(configFile string) (*Config, error) {
	var (
		config Config
		err    error
		reader *os.File
	)
	reader, err = os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	decoder := yaml.NewDecoder(reader)
	if err = decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func updateGrafanaService(panelItem *ConfigPanelItem, service string) {
	panelItem.Availability = toPtr(strings.Replace(*panelItem.Availability, "grafana.service", service, -1))
	panelItem.Requests = toPtr(strings.Replace(*panelItem.Requests, "grafana.service", service, -1))
	panelItem.Errors1 = toPtr(strings.Replace(*panelItem.Errors1, "grafana.service", service, -1))
	panelItem.Errors2 = toPtr(strings.Replace(*panelItem.Errors2, "grafana.service", service, -1))
	if panelItem.Errors3 != nil {
		panelItem.Errors3 = toPtr(strings.Replace(*panelItem.Errors3, "grafana.service", service, -1))
	}
	panelItem.Duration = toPtr(strings.Replace(*panelItem.Duration, "grafana.service", service, -1))
	panelItem.Latency = toPtr(strings.Replace(*panelItem.Latency, "grafana.service", service, -1))
}

// Keys creates an array of the map keys.
func keys[K comparable, V any](in map[K]V) []K {
	result := make([]K, 0, len(in))

	for k := range in {
		result = append(result, k)
	}

	return result
}
