package configuration

import (
	"math"
	"net/url"
	"reflect"
	"sort"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/discovery"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/httputil"
	"golang.org/x/exp/constraints"
)

var Empty = &Config{
	FlushInterval:             time.Duration(math.MaxInt64),
	DefaultCollectionInterval: time.Duration(math.MaxInt64),
	SinkExportDataTimeout:     time.Duration(math.MinInt64),
	DiscoveryConfig: discovery.Config{
		DiscoveryInterval: time.Duration(math.MaxInt64),
	},
}

func Combine(a, b *Config) *Config {
	return &Config{
		FlushInterval:             combineRecurringIntervals(a.FlushInterval, b.FlushInterval),
		DefaultCollectionInterval: combineRecurringIntervals(a.DefaultCollectionInterval, b.DefaultCollectionInterval),
		SinkExportDataTimeout:     max(a.SinkExportDataTimeout, b.SinkExportDataTimeout),
		EnableDiscovery:           a.EnableDiscovery || b.EnableDiscovery,
		EnableEvents:              a.EnableEvents || b.EnableEvents,
		ClusterName:               max(a.ClusterName, b.ClusterName),
		Sinks:                     combineSinks(a.Sinks, b.Sinks),
		Sources:                   combineNilable(combineSources)(a.Sources, b.Sources),
		EventsConfig:              combineEventsConfig(a.EventsConfig, b.EventsConfig),
		DiscoveryConfig:           combineDiscoveryConfig(a.DiscoveryConfig, b.DiscoveryConfig),
		OmitBucketSuffix:          a.OmitBucketSuffix || b.OmitBucketSuffix,
		Experimental:              combineStringSet(a.Experimental, b.Experimental),
	}
}

func combineRecurringIntervals(a, b time.Duration) time.Duration {
	return min(a, b)
}

func combineDiscoveryConfig(a, b discovery.Config) discovery.Config {
	return discovery.Config{
		DiscoveryInterval:          min(a.DiscoveryInterval, b.DiscoveryInterval),
		AnnotationPrefix:           max(a.AnnotationPrefix, b.AnnotationPrefix),
		AnnotationExcludes:         combineSelectorSets(a.AnnotationExcludes, b.AnnotationExcludes),
		EnableRuntimePlugins:       a.EnableRuntimePlugins || b.EnableRuntimePlugins,
		DisableAnnotationDiscovery: a.DisableAnnotationDiscovery || b.DisableAnnotationDiscovery,
		PluginConfigs:              combinePluginConfigSets(a.PluginConfigs, b.PluginConfigs),
		Global:                     combineDiscoveryGlobalConfig(a.Global, b.Global),
		PromConfigs:                combineDiscoveryPrometheusConfig(a.PromConfigs, b.PromConfigs),
	}
}

var combineDiscoveryPrometheusConfig = combineSets[discovery.PrometheusConfig](
	func(a, b discovery.PrometheusConfig) bool {
		return a.Name < b.Name
	},
	func(a, b discovery.PrometheusConfig) bool {
		return a.Name == b.Name
	},
)

func combineDiscoveryGlobalConfig(a discovery.GlobalConfig, b discovery.GlobalConfig) discovery.GlobalConfig {
	return discovery.GlobalConfig{
		DiscoveryInterval: max(a.DiscoveryInterval, b.DiscoveryInterval),
	}
}

var combinePluginConfigSets = combineSets[discovery.PluginConfig](
	func(a, b discovery.PluginConfig) bool {
		return a.Name < b.Name
	},
	func(a, b discovery.PluginConfig) bool {
		return a.Name == b.Name
	},
)

func combineSets[T any](less func(T, T) bool, equal func(T, T) bool) func([]T, []T) []T {
	return combineSetsWithCombinableElements(less, equal, func(a, b T) T {
		return a
	})
}

func combineSetsWithCombinableElements[T any](less func(T, T) bool, equal func(T, T) bool, combineElement func(T, T) T) func([]T, []T) []T {
	return func(a, b []T) []T {
		var c []T
		if len(a) > 0 {
			c = make([]T, len(a))
			copy(c, a)
		}
		for _, bElem := range b {
			found := false
			for i, aElem := range a {
				if equal(aElem, bElem) {
					c[i] = combineElement(aElem, bElem)
					found = true
					break
				}
			}
			if !found {
				c = append(c, bElem)
			}
		}
		sort.Slice(c, func(i, j int) bool {
			return less(c[i], c[j])
		})
		return c
	}
}

func deepEq[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

func setIsLess[T constraints.Ordered](a, b []T) bool {
	aCopy := make([]T, len(a))
	copy(aCopy, a)
	sort.Slice(aCopy, func(i, j int) bool {
		return aCopy[i] < aCopy[j]
	})
	bCopy := make([]T, len(b))
	copy(bCopy, b)
	sort.Slice(bCopy, func(i, j int) bool {
		return bCopy[i] < bCopy[j]
	})
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i] >= b[i] {
			return false
		}
	}
	return true
}

func labelsMapIsLess(a, b map[string][]string) bool {
	oneIsStrictlyLess := false
	for k := range a {
		if setIsLess(b[k], a[k]) {
			return false
		}
		if setIsLess(a[k], b[k]) {
			oneIsStrictlyLess = true
		}
	}
	for k := range b {
		if setIsLess(b[k], a[k]) {
			return false
		}
		if setIsLess(a[k], b[k]) {
			oneIsStrictlyLess = true
		}
	}
	return oneIsStrictlyLess
}

var combineSelectorSets = combineSets[discovery.Selectors](
	func(a, b discovery.Selectors) bool {
		return a.ResourceType < b.ResourceType &&
			setIsLess(a.Images, b.Images) &&
			labelsMapIsLess(a.Labels, b.Labels) &&
			setIsLess(a.Namespaces, b.Namespaces)
	},
	deepEq[discovery.Selectors],
)

func combineEventsConfig(a, b EventsConfig) EventsConfig {
	return EventsConfig{
		Filters:     combineEventsFilter(a.Filters, b.Filters),
		ClusterName: max(a.ClusterName, b.ClusterName),
		ClusterUUID: max(a.ClusterUUID, b.ClusterUUID),
	}
}

func combineEventsFilter(a, b EventsFilter) EventsFilter {
	return EventsFilter{
		TagAllowList:     combineTagSets(a.TagAllowList, b.TagAllowList),
		TagDenyList:      combineTagSets(a.TagDenyList, b.TagDenyList),
		TagAllowListSets: combineTagSetLists(a.TagAllowListSets, b.TagAllowListSets),
		TagDenyListSets:  combineTagSetLists(a.TagDenyListSets, b.TagDenyListSets),
		TagWhitelist:     combineTagSets(a.TagWhitelist, b.TagWhitelist),
		TagBlacklist:     combineTagSets(a.TagBlacklist, b.TagBlacklist),
		TagWhitelistSets: combineTagSetLists(a.TagWhitelistSets, b.TagWhitelistSets),
		TagBlacklistSets: combineTagSetLists(a.TagBlacklistSets, b.TagBlacklistSets),
	}
}

func combineSources(a, b SourceConfig) SourceConfig {
	return SourceConfig{
		SummaryConfig:     combineNilable(combineSummarySourceConfigs)(a.SummaryConfig, b.SummaryConfig),
		CadvisorConfig:    combineNilable(combineCadvisorConfigs)(a.CadvisorConfig, b.CadvisorConfig),
		PrometheusConfigs: combinePrometheusConfigSets(a.PrometheusConfigs, b.PrometheusConfigs),
		TelegrafConfigs:   combineTelegrafConfigSets(a.TelegrafConfigs, b.TelegrafConfigs),
		SystemdConfig:     combineNilable(combineSystemdConfigs)(a.SystemdConfig, b.SystemdConfig),
		StatsConfig:       combineNilable(combineStatsConfigs)(a.StatsConfig, b.StatsConfig),
		StateConfig:       combineNilable(combineKubernetesStateSourceConfigs)(a.StateConfig, b.StateConfig),
	}
}

func combineKubernetesStateSourceConfigs(a, b KubernetesStateSourceConfig) KubernetesStateSourceConfig {
	return KubernetesStateSourceConfig{
		Transforms: combineTransforms(a.Transforms, b.Transforms),
		Collection: combineCollectionConfig(a.Collection, b.Collection),
	}
}

func combineStatsConfigs(a, b StatsSourceConfig) StatsSourceConfig {
	return StatsSourceConfig{
		Transforms: combineTransforms(a.Transforms, b.Transforms),
		Collection: combineCollectionConfig(a.Collection, b.Collection),
	}
}

func combineSystemdConfigs(a, b SystemdSourceConfig) SystemdSourceConfig {
	return SystemdSourceConfig{
		Transforms:              combineTransforms(a.Transforms, b.Transforms),
		Collection:              combineCollectionConfig(a.Collection, b.Collection),
		IncludeTaskMetrics:      a.IncludeTaskMetrics || b.IncludeTaskMetrics,
		IncludeStartTimeMetrics: a.IncludeStartTimeMetrics || b.IncludeStartTimeMetrics,
		IncludeRestartMetrics:   a.IncludeRestartMetrics || b.IncludeRestartMetrics,
		UnitAllowList:           combineStringSet(a.UnitAllowList, b.UnitAllowList),
		UnitDenyList:            combineStringSet(a.UnitAllowList, b.UnitAllowList),
	}
}

func tagsIsLess(a, b map[string]string) bool {
	oneStrictlyLess := false
	for k := range a {
		if a[k] > b[k] {
			return false
		}
		if a[k] < b[k] {
			oneStrictlyLess = true
		}
	}
	for k := range b {
		if a[k] > b[k] {
			return false
		}
		if a[k] < b[k] {
			oneStrictlyLess = true
		}
	}
	return oneStrictlyLess
}

func transformsIsLess(a, b Transforms) bool {
	return a.Source < b.Source &&
		a.Prefix < b.Prefix &&
		tagsIsLess(a.Tags, b.Tags) &&
		filterIsLess(a.Filters, b.Filters) &&
		(!a.ConvertHistograms && b.ConvertHistograms)
}

func filterIsLess(a, b filter.Config) bool {
	return setIsLess(a.MetricAllowList, b.MetricAllowList) &&
		setIsLess(a.MetricDenyList, b.MetricDenyList) &&
		labelsMapIsLess(a.MetricTagAllowList, b.MetricTagAllowList) &&
		labelsMapIsLess(a.MetricTagDenyList, b.MetricTagDenyList) &&
		setIsLess(a.TagInclude, b.TagInclude) &&
		setIsLess(a.TagExclude, b.TagInclude) &&
		setIsLess(a.TagGuaranteeList, b.TagGuaranteeList) &&
		setIsLess(a.MetricWhitelist, b.MetricWhitelist) &&
		setIsLess(a.MetricBlacklist, b.MetricBlacklist) &&
		labelsMapIsLess(a.MetricTagWhitelist, b.MetricTagWhitelist) &&
		labelsMapIsLess(a.MetricTagBlacklist, b.MetricTagBlacklist)
}

var combineTelegrafConfigSets = combineSetsWithCombinableElements[*TelegrafSourceConfig](
	func(a, b *TelegrafSourceConfig) bool {
		return transformsIsLess(a.Transforms, b.Transforms) &&
			collectionConfigIsLess(a.Collection, b.Collection) &&
			setIsLess(a.Plugins, b.Plugins) &&
			a.Conf < b.Conf
	},
	func(a, b *TelegrafSourceConfig) bool {
		return reflect.DeepEqual(a.Transforms, b.Transforms) &&
			reflect.DeepEqual(a.Collection, b.Collection) &&
			reflect.DeepEqual(a.Plugins, b.Plugins) &&
			a.Conf == b.Conf
	},
	combineTelegrafConfigs,
)

func collectionConfigIsLess(a, b CollectionConfig) bool {
	return a.Interval < b.Interval && a.Timeout < b.Timeout
}

func combineTelegrafConfigs(a, b *TelegrafSourceConfig) *TelegrafSourceConfig {
	return &TelegrafSourceConfig{
		Transforms: combineTransforms(a.Transforms, b.Transforms),
		Collection: combineCollectionConfig(a.Collection, b.Collection),
		Plugins:    combineStringSet(a.Plugins, b.Plugins),
		Conf:       max(a.Conf, b.Conf),
	}
}

var combinePrometheusConfigSets = combineSetsWithCombinableElements[*PrometheusSourceConfig](
	func(a, b *PrometheusSourceConfig) bool {
		return a.URL < b.URL
	},
	func(a, b *PrometheusSourceConfig) bool {
		return a.URL == b.URL
	},
	combinePrometheusSourceConfigs,
)

func combinePrometheusSourceConfigs(a, b *PrometheusSourceConfig) *PrometheusSourceConfig {
	return &PrometheusSourceConfig{
		Transforms:        combineTransforms(a.Transforms, b.Transforms),
		Collection:        combineCollectionConfig(a.Collection, b.Collection),
		URL:               max(a.URL, b.URL),
		HTTPClientConfig:  combineHTTPClientConfig(a.HTTPClientConfig, b.HTTPClientConfig),
		Name:              max(a.Name, b.Name),
		Discovered:        max(a.Discovered, b.Discovered),
		UseLeaderElection: a.UseLeaderElection || b.UseLeaderElection,
	}
}

func combineHTTPClientConfig(a httputil.ClientConfig, b httputil.ClientConfig) httputil.ClientConfig {
	return httputil.ClientConfig{
		BearerToken:     max(a.BearerToken, b.BearerToken),
		BearerTokenFile: max(a.BearerTokenFile, b.BearerTokenFile),
		ProxyURL:        combineUtilURLs(a.ProxyURL, b.ProxyURL),
		TLSConfig:       httputil.TLSConfig{},
	}
}

func combineUtilURLs(a, b httputil.URL) httputil.URL {
	return httputil.URL{
		URL: combineNilable[url.URL](combineURLs)(a.URL, b.URL),
	}
}

func combineURLs(a, b url.URL) url.URL {
	return url.URL{
		Scheme:      max(a.Scheme, b.Scheme),
		Opaque:      max(a.Opaque, b.Opaque),
		User:        combineNilable[url.Userinfo](combineUserInfos)(a.User, b.User),
		Host:        max(a.Host, b.Host),
		Path:        max(a.Path, b.Path),
		RawPath:     max(a.RawPath, b.RawPath),
		OmitHost:    a.OmitHost || b.OmitHost,
		ForceQuery:  a.ForceQuery || b.ForceQuery,
		RawQuery:    max(a.RawQuery, b.RawQuery),
		Fragment:    max(a.Fragment, b.Fragment),
		RawFragment: max(a.RawFragment, b.RawFragment),
	}

}

func combineUserInfos(a, b url.Userinfo) url.Userinfo {
	username := max(a.Username(), b.Username())
	aPassword, aPasswordSet := a.Password()
	bPassword, bPasswordSet := b.Password()
	if aPasswordSet || bPasswordSet {
		return *url.UserPassword(username, max(aPassword, bPassword))
	}
	return *url.User(username)
}

func combineNilable[T any](combine func(T, T) T) func(a, b *T) *T {
	return func(a, b *T) *T {
		if a == nil {
			return b
		}
		if b == nil {
			return a
		}
		c := combine(*a, *b)
		return &c
	}
}

func combineCadvisorConfigs(a, b CadvisorSourceConfig) CadvisorSourceConfig {
	return CadvisorSourceConfig{
		Transforms: combineTransforms(a.Transforms, b.Transforms),
		Collection: combineCollectionConfig(a.Collection, b.Collection),
	}
}

func combineSummarySourceConfigs(a, b SummarySourceConfig) SummarySourceConfig {
	return SummarySourceConfig{
		Transforms:        combineTransforms(a.Transforms, b.Transforms),
		Collection:        combineCollectionConfig(a.Collection, b.Collection),
		URL:               max(a.URL, b.URL),
		KubeletPort:       max(a.KubeletPort, b.KubeletPort),
		KubeletHttps:      max(a.KubeletHttps, b.KubeletHttps),
		InClusterConfig:   max(a.InClusterConfig, b.InClusterConfig),
		UseServiceAccount: max(a.UseServiceAccount, b.UseServiceAccount),
		Insecure:          max(a.Insecure, b.Insecure),
		Auth:              max(a.Auth, b.Auth),
	}
}

func combineCollectionConfig(a, b CollectionConfig) CollectionConfig {
	return CollectionConfig{
		Interval: max(a.Interval, b.Interval),
		Timeout:  max(a.Timeout, b.Timeout),
	}
}

func boolPtrIsLess(a, b *bool) bool {
	if b == nil {
		return false
	}
	if a == nil {
		return true
	}
	return !*a && *b
}

var combineSinks = combineSetsWithCombinableElements[*SinkConfig](
	func(a *SinkConfig, b *SinkConfig) bool {
		return transformsIsLess(a.Transforms, b.Transforms) &&
			a.Type < b.Type &&
			boolPtrIsLess(a.EnableEvents, b.EnableEvents) &&
			a.Server < b.Server &&
			a.Token < b.Token &&
			a.ProxyAddress < b.ProxyAddress &&
			a.ExternalEndpointURL < b.ExternalEndpointURL
	},
	func(a, b *SinkConfig) bool {
		return reflect.DeepEqual(a.Transforms, b.Transforms) &&
			a.Type == b.Type &&
			a.EnableEvents == b.EnableEvents &&
			a.Server == b.Server &&
			a.Token == b.Token &&
			a.ProxyAddress == b.ProxyAddress &&
			a.ExternalEndpointURL == b.ExternalEndpointURL
	},
	combineSinkConfig,
)

func combineSinkConfig(a, b *SinkConfig) *SinkConfig {
	return &SinkConfig{
		Type:                      max(a.Type, b.Type),
		EnableEvents:              combineBoolPtrs(a.EnableEvents, b.EnableEvents),
		Server:                    max(a.Server, b.Server),
		Token:                     max(a.Token, b.Token),
		ProxyAddress:              max(a.ProxyAddress, b.ProxyAddress),
		ExternalEndpointURL:       max(a.ExternalEndpointURL, b.ExternalEndpointURL),
		Transforms:                combineTransforms(a.Transforms, b.Transforms),
		BatchSize:                 max(a.BatchSize, b.BatchSize),
		MaxBufferSize:             max(a.MaxBufferSize, b.MaxBufferSize),
		TestMode:                  a.TestMode || b.TestMode,
		ErrorLogPercent:           max(a.ErrorLogPercent, b.ErrorLogPercent),
		ExternalEndpointAccessKey: max(a.ExternalEndpointAccessKey, b.ExternalEndpointAccessKey),
		ClusterName:               max(a.ClusterName, b.ClusterName),
		InternalStatsPrefix:       max(a.InternalStatsPrefix, b.InternalStatsPrefix),
		HeartbeatInterval:         min(a.HeartbeatInterval, b.HeartbeatInterval),
	}
}

func combineBoolPtrs(a, b *bool) *bool {
	c := defaultBoolPtr(a) || defaultBoolPtr(b)
	return &c
}

func defaultBoolPtr(a *bool) bool {
	if a != nil {
		return *a
	}
	return false
}

func combineTransforms(a, b Transforms) Transforms {
	return Transforms{
		Source:            max(a.Source, b.Source),
		Prefix:            max(a.Prefix, b.Prefix),
		Tags:              combineTags(a.Tags, b.Tags),
		Filters:           combineFilterConfig(a.Filters, b.Filters),
		ConvertHistograms: a.ConvertHistograms || b.ConvertHistograms,
	}
}

func combineFilterConfig(a, b filter.Config) filter.Config {
	return filter.Config{
		MetricAllowList:    combineStringSet(a.MetricAllowList, b.MetricAllowList),
		MetricDenyList:     combineStringSet(a.MetricDenyList, b.MetricDenyList),
		MetricTagAllowList: combineTagSets(a.MetricTagAllowList, b.MetricTagAllowList),
		MetricTagDenyList:  combineTagSets(a.MetricTagDenyList, b.MetricTagDenyList),
		TagInclude:         combineStringSet(a.TagInclude, b.TagInclude),
		TagExclude:         combineStringSet(a.TagExclude, b.TagExclude),
		TagGuaranteeList:   combineStringSet(a.TagGuaranteeList, b.TagGuaranteeList),
		MetricWhitelist:    combineStringSet(a.MetricWhitelist, b.MetricWhitelist),
		MetricBlacklist:    combineStringSet(a.MetricBlacklist, b.MetricBlacklist),
		MetricTagWhitelist: combineTagSets(a.MetricTagWhitelist, b.MetricTagWhitelist),
		MetricTagBlacklist: combineTagSets(a.MetricTagBlacklist, b.MetricTagBlacklist),
	}
}

var combineTagSetLists = combineSets[map[string][]string](
	labelsMapIsLess,
	func(a, b map[string][]string) bool {
		return reflect.DeepEqual(a, b)
	},
)

var combineTags = combineMaps[string, string](max[string])
var combineTagSets = combineMaps[string, []string](combineStringSet)

func combineMaps[K comparable, V any](combineValue func(V, V) V) func(a, b map[K]V) map[K]V {
	return func(a, b map[K]V) map[K]V {
		if len(a) == 0 && len(b) == 0 {
			return nil
		}
		c := map[K]V{}
		for k := range a {
			c[k] = combineValue(a[k], b[k])
		}
		for k := range b {
			c[k] = combineValue(a[k], b[k])
		}
		return c
	}
}

var combineStringSet = combineSets[string](
	func(a string, b string) bool {
		return a < b
	},
	deepEq[string],
)

func max[T constraints.Ordered](a, b T) T {
	if a >= b {
		return a
	}
	return b
}

func min[T constraints.Ordered](a, b T) T {
	if a <= b {
		return a
	}
	return b
}
