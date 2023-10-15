package generate

type Exception struct {
	Path    string  `json:"-" yaml:"path"`
	Method  string  `json:"-" yaml:"method"`
	Errors3 *string `json:"-" yaml:"errors3"`
}

type ConfigPanelItem struct {
	Path         *string      `yaml:"path" json:"path,omitempty"`
	Type         *string      `yaml:"type" json:"type,omitempty"`
	Availability *string      `yaml:"availability" json:"availability,omitempty"`
	Requests     *string      `yaml:"requests" json:"requests,omitempty"`
	Errors1      *string      `yaml:"errors1" json:"errors1,omitempty"`
	Errors2      *string      `yaml:"errors2" json:"errors2,omitempty"`
	Errors3      *string      `yaml:"errors3" json:"errors3,omitempty"`
	Duration     *string      `yaml:"duration" json:"duration,omitempty"`
	Latency      *string      `yaml:"latency" json:"latency,omitempty"`
	Exception    *Exception   `yaml:"exception" json:"exception,omitempty"`
	Panels       *[]PanelItem `yaml:"panels" json:"panels,omitempty"`
}

type GrafanaMetadataLabels struct {
	GrafanaDashboard string `yaml:"grafana_dashboard" json:"grafana_dashboard"`
}

type GrafanaMetadataAnnotations struct {
	GrafanaFolder string `yaml:"grafana_folder" json:"grafana_folder"`
}

type GrafanaMetadata struct {
	Name        string                      `yaml:"name" json:"name"`
	Labels      *GrafanaMetadataLabels      `yaml:"labels" json:"labels,omitempty"`
	Annotations *GrafanaMetadataAnnotations `yaml:"annotations" json:"annotations,omitempty"`
}

type GrafanaConfig struct {
	Uid                *string          `yaml:"uid" json:"uid,omitempty"`
	Service            string           `yaml:"service" json:"service,omitempty"`
	SkipServiceAccount *bool            `yaml:"skipServiceAccount" json:"skip_service_account,omitempty"`
	Title              *string          `yaml:"title" json:"title,omitempty"`
	Style              *string          `yaml:"style" json:"style,omitempty"`
	SchemaVersion      *int             `yaml:"schemaVersion" json:"schemaVersion,omitempty"`
	Version            *int             `yaml:"version" json:"version,omitempty"`
	ApiVersion         string           `yaml:"apiVersion" json:"apiVersion,omitempty"`
	Timezone           *string          `yaml:"timezone" json:"timezone,omitempty"`
	Metadata           *GrafanaMetadata `yaml:"metadata" json:"metadata,omitempty"`
}

type Requires struct {
	Id      *string `json:"id,omitempty" yaml:"id"`
	Type    *string `json:"type,omitempty" yaml:"type"`
	Name    *string `json:"name,omitempty" yaml:"name"`
	Version *string `json:"version,omitempty" yaml:"version"`
}

type Config struct {
	Time            *Time              `yaml:"time" json:"time,omitempty"`
	Timepicker      *Timepicker        `yaml:"timepicker" json:"timepicker,omitempty"`
	Templating      *Template          `yaml:"templating" json:"templating,omitempty"`
	Annotations     *AnnotationList    `yaml:"annotations" json:"annotations,omitempty"`
	PanelDatasource *Datasource        `yaml:"panelDatasource" json:"panelDatasource,omitempty"`
	Grafana         *GrafanaConfig     `yaml:"grafana" json:"grafana,omitempty"`
	Requires        *[]Requires        `yaml:"__requires" json:"requires,omitempty"`
	Panels          *[]ConfigPanelItem `yaml:"panels" json:"panels,omitempty"`
}

type TextValue struct {
	Selected *bool   `json:"selected,omitempty" yaml:"selected"`
	Text     *string `json:"text,omitempty" yaml:"text"`
	Value    *string `json:"value,omitempty" yaml:"value"`
}

type GridPos struct {
	H int `json:"h" yaml:"h"`
	W int `json:"w" yaml:"w"`
	X int `json:"x" yaml:"x"`
	Y int `json:"y" yaml:"y"`
}

type Datasource struct {
	Type *string `json:"type,omitempty" yaml:"type,omitempty"`
	Uid  *string `json:"uid,omitempty" yaml:"uid,omitempty"`
}

type Target struct {
	Expr           string  `json:"expr" yaml:"expr,omitempty"`
	Format         *string `json:"format,omitempty" yaml:"format,omitempty"`
	Instant        *bool   `json:"instant,omitempty" yaml:"instant,omitempty"`
	Hide           *bool   `json:"hide,omitempty" yaml:"hide,omitempty"`
	LegendFormat   *string `json:"legendFormat,omitempty" yaml:"legendFormat,omitempty"`
	Interval       *string `json:"interval,omitempty" yaml:"interval,omitempty"`
	IntervalFactor *int    `json:"intervalFactor,omitempty" yaml:"intervalFactor,omitempty"`
	RefId          string  `json:"refId" yaml:"refId,omitempty"`
}

type Color struct {
	Mode *string `json:"mode,omitempty" yaml:"mode,omitempty"`
}

type Result struct {
	Text *string `json:"text,omitempty" yaml:"text,omitempty"`
}

type MappingOptions struct {
	Match  *string `json:"match,omitempty" yaml:"match,omitempty"`
	Result *Result `json:"result,omitempty" yaml:"result,omitempty"`
}

type Mapping struct {
	Options *MappingOptions `json:"options,omitempty" yaml:"options,omitempty"`
	Type    *string         `json:"type,omitempty" yaml:"type,omitempty"`
}

type Step struct {
	Color *string  `json:"color,omitempty" yaml:"color,omitempty"`
	Value *float32 `json:"value,omitempty" yaml:"value,omitempty"`
}

type Threshold struct {
	Mode  *string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Steps *[]Step `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Custom struct {
	Filterable bool `json:"filterable,omitempty" yaml:"filterable,omitempty"`
}

type Defaults struct {
	Decimals   *int       `json:"decimals,omitempty" yaml:"decimals,omitempty"`
	Color      *Color     `json:"color,omitempty" yaml:"color,omitempty"`
	Mappings   *[]Mapping `json:"mappings,omitempty" yaml:"mappings,omitempty"`
	Thresholds *Threshold `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`
	Unit       *string    `json:"unit,omitempty" yaml:"unit,omitempty"`
	Overrides  *[]string  `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Custom     *Custom    `json:"custom,omitempty" yaml:"custom,omitempty"`
}

type FieldConfig struct {
	Defaults  *Defaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Overrides *[]string `json:"overrides,omitempty" yaml:"overrides,omitempty"`
}

type ReduceOptions struct {
	Calcs  *[]string `json:"calcs,omitempty" yaml:"calcs,omitempty"`
	Fields *string   `json:"fields,omitempty" yaml:"fields,omitempty"`
	Values *bool     `json:"values,omitempty" yaml:"values,omitempty"`
}

type FieldOptions struct {
	Calcs     *[]string `json:"calcs,omitempty" yaml:"calcs,omitempty"`
	Defaults  *Defaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Overrides *[]string `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Values    *bool     `json:"values,omitempty" yaml:"values,omitempty"`
}

type Options struct {
	ColorMode     *string        `json:"colorMode,omitempty" yaml:"colorMode,omitempty"`
	GraphMode     *string        `json:"graphMode,omitempty" yaml:"graphMode,omitempty"`
	JustifyMode   *string        `json:"justifyMode,omitempty" yaml:"justifyMode,omitempty"`
	Orientation   *string        `json:"orientation,omitempty" yaml:"orientation,omitempty"`
	ReduceOptions *ReduceOptions `json:"reduceOptions,omitempty" yaml:"reduceOptions,omitempty"`
	FieldOptions  *FieldOptions  `json:"fieldOptions,omitempty" yaml:"fieldOptions,omitempty"`
	Text          *struct {
	} `json:"text,omitempty" yaml:"text,omitempty"`
	TextMode             *string `json:"textMode,omitempty" yaml:"textMode,omitempty"`
	AlertThreshold       *bool   `json:"alertThreshold,omitempty" yaml:"alertThreshold,omitempty"`
	ShowThresholdLabels  *bool   `json:"showThresholdLabels,omitempty" yaml:"showThresholdLabels,omitempty"`
	ShowThresholdMarkers *bool   `json:"showThresholdMarkers,omitempty" yaml:"showThresholdMarkers,omitempty"`
}

type Legend struct {
	AlignAsTable *bool `json:"alignAsTable,omitempty" yaml:"alignAsTable,omitempty"`
	Avg          *bool `json:"avg,omitempty" yaml:"avg,omitempty"`
	Current      *bool `json:"current,omitempty" yaml:"current,omitempty"`
	HideZero     *bool `json:"hideZero,omitempty" yaml:"hideZero,omitempty"`
	Max          *bool `json:"max,omitempty" yaml:"max,omitempty"`
	Min          *bool `json:"min,omitempty" yaml:"min,omitempty"`
	RightSide    *bool `json:"rightSide,omitempty" yaml:"rightSide,omitempty"`
	Show         *bool `json:"show,omitempty" yaml:"show,omitempty"`
	Total        *bool `json:"total,omitempty" yaml:"total,omitempty"`
	Values       *bool `json:"values,omitempty" yaml:"values,omitempty"`
}

type Tooltip struct {
	Shared        *bool   `json:"shared,omitempty" yaml:"shared,omitempty"`
	Sort          *int    `json:"sort,omitempty" yaml:"sort,omitempty"`
	ValueType     *string `json:"valueType,omitempty" yaml:"valueType,omitempty"`
	ShowHistogram *bool   `json:"showHistogram,omitempty" yaml:"showHistogram,omitempty"`
}

type XAxis struct {
	Mode   *string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Show   *bool     `json:"show,omitempty" yaml:"show,omitempty"`
	Values *[]string `json:"values,omitempty" yaml:"values,omitempty"`
}

type YAxe struct {
	Decimals *int    `json:"decimals,omitempty" yaml:"decimals,omitempty"`
	Format   *string `json:"format,omitempty" yaml:"format,omitempty"`
	Label    *string `json:"label,omitempty" yaml:"label,omitempty"`
	LogBase  *int    `json:"logBase,omitempty" yaml:"logBase,omitempty"`
	Min      *string `json:"min,omitempty" yaml:"min,omitempty"`
	Show     *bool   `json:"show,omitempty" yaml:"show,omitempty"`
}

type YAxis struct {
	Align   *bool   `json:"align,omitempty" yaml:"align,omitempty"`
	LogBase *int    `json:"logBase,omitempty" yaml:"logBase,omitempty"`
	Format  *string `json:"format,omitempty" yaml:"format,omitempty"`
}

type SeriesOverride struct {
	Alias       *string `json:"alias,omitempty" yaml:"alias,omitempty"`
	Color       *string `json:"color,omitempty" yaml:"color,omitempty"`
	LineWidth   *int    `json:"linewidth,omitempty" yaml:"lineWidth,omitempty"`
	DashLength  *int    `json:"dashLength,omitempty" yaml:"dashLength,omitempty"`
	Dash        *int    `json:"dash,omitempty" yaml:"dash,omitempty"`
	Dashes      *bool   `json:"dashes,omitempty" yaml:"dashes,omitempty"`
	SpaceLength *int    `json:"spaceLength,omitempty" yaml:"spaceLength,omitempty"`
	Yaxis       *int    `json:"yaxis,omitempty" yaml:"yaxis,omitempty"`
}

type PanelItem struct {
	AliasColors      *map[string]string `json:"aliasColors,omitempty" yaml:"aliasColors,omitempty"`
	Bars             *bool              `json:"bars,omitempty" yaml:"bars,omitempty"`
	DashLengths      *int               `json:"dashLengths,omitempty" yaml:"dashLengths,omitempty"`
	Dashes           *bool              `json:"dashes,omitempty" yaml:"dashes,omitempty"`
	Fill             *int               `json:"fill,omitempty" yaml:"fill,omitempty"`
	FillGradient     *int               `json:"fillGradient,omitempty" yaml:"fillGradient,omitempty"`
	HiddenSeries     *bool              `json:"hiddenSeries,omitempty" yaml:"hiddenSeries,omitempty"`
	Legend           *Legend            `json:"legend,omitempty" yaml:"legend,omitempty"`
	Lines            *bool              `json:"lines,omitempty" yaml:"lines,omitempty"`
	LineWidth        *int               `json:"lineWidth,omitempty" yaml:"lineWidth,omitempty"`
	NullPointMode    *string            `json:"nullPointMode,omitempty" yaml:"nullPointMode,omitempty"`
	Datasource       *Datasource        `json:"datasource,omitempty" yaml:"datasource,omitempty"`
	Description      *string            `json:"description,omitempty" yaml:"description,omitempty"`
	FieldConfig      *FieldConfig       `json:"fieldConfig,omitempty" yaml:"fieldConfig,omitempty"`
	GridPos          GridPos            `json:"gridPos,omitempty" yaml:"gridPos,omitempty"`
	Id               int                `json:"id,omitempty" yaml:"id,omitempty"`
	Links            *[]string          `json:"Links,omitempty" yaml:"Links,omitempty"`
	MaxDataPoints    *int               `json:"maxDataPoints,omitempty" yaml:"maxDataPoints,omitempty"`
	Options          *Options           `json:"options,omitempty" yaml:"options,omitempty"`
	PluginVersion    *string            `json:"pluginVersion,omitempty" yaml:"pluginVersion,omitempty"`
	TimeShift        *string            `json:"timeShift,omitempty" yaml:"timeShift,omitempty"`
	Targets          []Target           `json:"targets,omitempty" yaml:"targets,omitempty"`
	Title            *string            `json:"title,omitempty" yaml:"title,omitempty"`
	Type             *string            `json:"type,omitempty" yaml:"type,omitempty"`
	Percentage       *bool              `json:"percentage,omitempty" yaml:"percentage,omitempty"`
	HideTimeOverride *bool              `json:"hideTimeOverride,omitempty" yaml:"hideTimeOverride,omitempty"`
	PointRadius      *int               `json:"pointRadius,omitempty" yaml:"pointRadius,omitempty"`
	Points           *bool              `json:"points,omitempty" yaml:"points,omitempty"`
	Renderer         *string            `json:"renderer,omitempty" yaml:"renderer,omitempty"`
	SeriesOverrides  *[]SeriesOverride  `json:"seriesOverrides,omitempty" yaml:"seriesOverrides,omitempty"`
	SpaceLength      *int               `json:"spaceLength,omitempty" yaml:"spaceLength,omitempty"`
	Stack            *bool              `json:"stack,omitempty" yaml:"stack,omitempty"`
	SteppedLine      *bool              `json:"steppedLine,omitempty" yaml:"steppedLine,omitempty"`
	Transparent      *bool              `json:"transparent,omitempty" yaml:"transparent,omitempty"`
	HideZeroBuckets  *bool              `json:"hideZeroBuckets,omitempty" yaml:"hideZeroBuckets,omitempty"`
	HighlightCards   *bool              `json:"highlightCards,omitempty" yaml:"highlightCards,omitempty"`
	ReverseBuckets   *bool              `json:"reverseBuckets,omitempty" yaml:"reverseBuckets,omitempty"`
	Thresholds       *[]Threshold       `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`
	TimeRegions      *[]struct{}        `json:"timeRegions,omitempty" yaml:"timeRegions,omitempty"`
	Tooltip          *Tooltip           `json:"tooltip,omitempty" yaml:"tooltip,omitempty"`
	XAxis            *XAxis             `json:"xaxis,omitempty" yaml:"xaxis,omitempty"`
	Yaxes            *[]YAxe            `json:"yaxes,omitempty" yaml:"yaxes,omitempty"`
	YAxis            *YAxis             `json:"yaxis,omitempty" yaml:"yaxis,omitempty"`
	Color            *Colors            `json:"color,omitempty" yaml:"color,omitempty"`
	DataFormat       string             `json:"dataFormat,omitempty" yaml:"dataFormat"`
	YBucketBound     string             `json:"YBucketBound,omitempty" yaml:"YBucketBound"`
}

type Colors struct {
	CardColor   *string  `json:"cardColor,omitempty" yaml:"cardColor"`
	ColorScale  *string  `json:"colorScale,omitempty" yaml:"colorScale"`
	ColorScheme *string  `json:"colorScheme,omitempty" yaml:"colorScheme"`
	Exponent    *float32 `json:"exponent,omitempty" yaml:"exponent"`
	Mode        *string  `json:"mode,omitempty" yaml:"mode"`
}

type Time struct {
	From *string `json:"from" yaml:"from"`
	To   *string `json:"to" yaml:"to"`
}

type Timepicker struct {
	RefreshIntervals *[]string `json:"refresh_intervals" yaml:"refreshIntervals"`
	TimeOptions      *[]string `json:"time_options" yaml:"timeOptions"`
}

type Query struct {
	Query *string `json:"query,omitempty" yaml:"query,omitempty"`
	RefId *string `json:"refId,omitempty" yaml:"refId,omitempty"`
}

type ListItem struct {
	AllValue       *string      `json:"allValue,omitempty" yaml:"allValue,omitempty"`
	DataSource     *Datasource  `json:"datasource,omitempty" yaml:"datasource,omitempty"`
	Definition     *string      `json:"definition,omitempty" yaml:"definition,omitempty"`
	Label          *string      `json:"label,omitempty" yaml:"label,omitempty"`
	Current        *TextValue   `json:"current,omitempty" yaml:"current,omitempty"`
	Hide           *int         `json:"hide,omitempty" yaml:"hide,omitempty"`
	IncludeAll     *bool        `json:"includeAll,omitempty" yaml:"includeAll,omitempty"`
	Multi          *bool        `json:"multi,omitempty" yaml:"multi,omitempty"`
	Name           *string      `json:"name,omitempty" yaml:"name,omitempty"`
	Options        *[]TextValue `json:"options,omitempty" yaml:"options,omitempty"`
	Query          *Query       `json:"query,omitempty" yaml:"query,omitempty"`
	Refresh        *int         `json:"refresh,omitempty" yaml:"refresh,omitempty"`
	Regex          *string      `json:"regex,omitempty" yaml:"regex,omitempty"`
	SkipUrlSync    *bool        `json:"skipUrlSync,omitempty" yaml:"skipUrlSync,omitempty"`
	Type           *string      `json:"type,omitempty" yaml:"type,omitempty"`
	Sort           *int         `json:"Sort,omitempty" yaml:"Sort,omitempty"`
	TagValuesQuery *string      `json:"tagValuesQuery,omitempty" yaml:"tagValuesQuery,omitempty"`
	TagsQuery      *string      `json:"tagsQuery,omitempty" yaml:"tagsQuery,omitempty"`
	UseTags        *bool        `json:"useTags,omitempty" yaml:"useTags,omitempty"`
}

type Template struct {
	List *[]ListItem `json:"list,omitempty" yaml:"list"`
}

type Panel struct {
	Type       string      `json:"type,omitempty" yaml:"type,omitempty"`
	Title      string      `json:"title,omitempty" yaml:"title,omitempty"`
	GridPos    GridPos     `json:"gridPos,omitempty" yaml:"gridPos,omitempty"`
	Id         int         `json:"id,omitempty" yaml:"id,omitempty"`
	Collapsed  bool        `json:"collapsed,omitempty" yaml:"collapsed,omitempty"`
	Datasource Datasource  `json:"datasource,omitempty" yaml:"datasource,omitempty"`
	Panels     []PanelItem `json:"panels" yaml:"panels,omitempty"`
}

type Grafana struct {
	Uid                  *string         `json:"uid,omitempty" yaml:"uid"`
	Title                *string         `json:"title,omitempty" yaml:"title"`
	Style                *string         `json:"style,omitempty" yaml:"style"`
	SchemaVersion        *int            `json:"schemaVersion,omitempty" yaml:"schemaVersion"`
	Version              *int            `json:"version,omitempty" yaml:"version"`
	Id                   *int            `json:"id,omitempty" yaml:"id"`
	Tags                 *[]string       `json:"tags,omitempty" yaml:"tags"`
	Timezone             *string         `json:"timezone,omitempty" yaml:"timezone"`
	Editable             *bool           `json:"editable,omitempty" yaml:"editable"`
	GraphTooltip         *int            `json:"graphTooltip,omitempty" yaml:"graphTooltip"`
	Links                *[]string       `json:"links,omitempty" yaml:"links"`
	FiscalYearStartMonth *int            `json:"fiscalYearStartMonth,omitempty" yaml:"fiscalYearStartMonth"`
	Iteration            *int64          `json:"iteration,omitempty" yaml:"iteration"`
	LiveNow              *bool           `json:"liveNow,omitempty" yaml:"liveNow"`
	WeekStart            *string         `json:"weekStart,omitempty" yaml:"weekStart"`
	Time                 *Time           `json:"time,omitempty" yaml:"time"`
	Timepicker           *Timepicker     `json:"timepicker,omitempty" yaml:"timepicker"`
	Templating           *Template       `json:"templating,omitempty" yaml:"templating"`
	Annotations          *AnnotationList `json:"annotations,omitempty" yaml:"annotations"`
	Requires             *[]Requires     `json:"__requires,omitempty"`
	Panels               []Panel         `json:"panels" yaml:"panels,omitempty"`
}

type AnnotationTarget struct {
	Limit    *int      `json:"limit,omitempty" yaml:"limit,omitempty"`
	MatchAny *bool     `json:"matchAny,omitempty" yaml:"matchAny,omitempty"`
	Tags     *[]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Type     *string   `json:"type,omitempty" yaml:"type,omitempty"`
}

type AnnotationItem struct {
	BuiltIn    *int              `json:"builtIn,omitempty" yaml:"builtin,omitempty"`
	Datasource *Datasource       `json:"datasource,omitempty" yaml:"datasource,omitempty"`
	Enable     *bool             `json:"enable,omitempty" yaml:"enable,omitempty"`
	Hide       *bool             `json:"hide,omitempty" yaml:"hide,omitempty"`
	IconColor  *string           `json:"iconColor,omitempty" yaml:"iconColor,omitempty"`
	Name       *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Target     *AnnotationTarget `json:"target,omitempty" yaml:"target,omitempty"`
	Type       *string           `json:"type,omitempty" yaml:"type,omitempty"`
}

type AnnotationList struct {
	List *[]AnnotationItem `json:"list,omitempty" yaml:"list"`
}
