package configuration

import (
	"net/url"
	"reflect"
	"sort"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/discovery"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/httputil"
	"golang.org/x/exp/constraints"
)

func Combine(a, b *Config) *Config {
	return &Config{
		FlushInterval:             combineFlushIntervals(a.FlushInterval, b.FlushInterval),
		DefaultCollectionInterval: max(a.DefaultCollectionInterval, b.DefaultCollectionInterval),
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

func combineFlushIntervals(a, b time.Duration) time.Duration {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	return min(a, b)
}

func combineDiscoveryConfig(a, b discovery.Config) discovery.Config {
	return discovery.Config{
		DiscoveryInterval:          max(a.DiscoveryInterval, b.DiscoveryInterval),
		AnnotationPrefix:           max(a.AnnotationPrefix, b.AnnotationPrefix),
		AnnotationExcludes:         combineSelectorSets(a.AnnotationExcludes, b.AnnotationExcludes),
		EnableRuntimePlugins:       a.EnableRuntimePlugins || b.EnableRuntimePlugins,
		DisableAnnotationDiscovery: a.DisableAnnotationDiscovery || b.DisableAnnotationDiscovery,
		PluginConfigs:              combinePluginConfigSets(a.PluginConfigs, b.PluginConfigs),
		Global:                     combineDiscoveryGlobalConfig(a.Global, b.Global),
		PromConfigs:                combineDiscoveryPrometheusConfig(a.PromConfigs, b.PromConfigs),
	}
}

var combineDiscoveryPrometheusConfig = combineSets[discovery.PrometheusConfig](func(a, b discovery.PrometheusConfig) bool {
	return a.Name == b.Name
})

func combineDiscoveryGlobalConfig(a discovery.GlobalConfig, b discovery.GlobalConfig) discovery.GlobalConfig {
	return discovery.GlobalConfig{
		DiscoveryInterval: max(a.DiscoveryInterval, b.DiscoveryInterval),
	}
}

var combinePluginConfigSets = combineSets[discovery.PluginConfig](func(a, b discovery.PluginConfig) bool {
	return a.Name == b.Name
})

func combineSets[T any](equal func(T, T) bool) func([]T, []T) []T {
	return combineSetsWithCombinableElements(equal, func(a, b T) T {
		return a
	})
}

func combineSetsWithCombinableElements[T any](equal func(T, T) bool, combineElement func(T, T) T) func([]T, []T) []T {
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
		return c
	}
}

func deepEq[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

var combineSelectorSets = combineSets[discovery.Selectors](deepEq[discovery.Selectors])

func combineEventsConfig(a, b EventsConfig) EventsConfig {
	return EventsConfig{
		Filters: combineEventsFilter(a.Filters, b.Filters),
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

var combineTelegrafConfigSets = combineSetsWithCombinableElements[*TelegrafSourceConfig](func(a, b *TelegrafSourceConfig) bool {
	return false
}, combineTelegrafConfigs)

func combineTelegrafConfigs(a, b *TelegrafSourceConfig) *TelegrafSourceConfig {
	return &TelegrafSourceConfig{
		Transforms: combineTransforms(a.Transforms, b.Transforms),
		Collection: combineCollectionConfig(a.Collection, b.Collection),
		Plugins:    combineStringSet(a.Plugins, b.Plugins),
		Conf:       max(a.Conf, b.Conf),
	}
}

var combinePrometheusConfigSets = combineSetsWithCombinableElements[*PrometheusSourceConfig](func(a, b *PrometheusSourceConfig) bool {
	return a.URL == b.URL
}, combinePrometheusSourceConfigs)

func combinePrometheusSourceConfigs(a, b *PrometheusSourceConfig) *PrometheusSourceConfig {
	return &PrometheusSourceConfig{
		Transforms:       combineTransforms(a.Transforms, b.Transforms),
		Collection:       combineCollectionConfig(a.Collection, b.Collection),
		URL:              max(a.URL, b.URL),
		HTTPClientConfig: combineHTTPClientConfig(a.HTTPClientConfig, b.HTTPClientConfig),
		Name:             max(a.Name, b.Name),
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

var combineSinks = combineSetsWithCombinableElements[*SinkConfig](func(a, b *SinkConfig) bool {
	return a.Type == b.Type && a.EnableEvents == b.EnableEvents &&
		a.Server == b.Server && a.Token == b.Token &&
		a.ProxyAddress == b.ProxyAddress &&
		a.ExternalEndpointURL == b.ExternalEndpointURL
}, combineSinkConfig)

func combineSinkConfig(a, b *SinkConfig) *SinkConfig {
	return &SinkConfig{
		Type:                max(a.Type, b.Type),
		EnableEvents:        combineBoolPtrs(a.EnableEvents, b.EnableEvents),
		Server:              max(a.Server, b.Server),
		Token:               max(a.Token, b.Token),
		ProxyAddress:        max(a.ProxyAddress, b.ProxyAddress),
		ExternalEndpointURL: max(a.ExternalEndpointURL, b.ExternalEndpointURL),
		Transforms:          combineTransforms(a.Transforms, b.Transforms),
		BatchSize:           max(a.BatchSize, b.BatchSize),
		MaxBufferSize:       max(a.MaxBufferSize, b.MaxBufferSize),
		TestMode:            a.TestMode || b.TestMode,
		ErrorLogPercent:     max(a.ErrorLogPercent, b.ErrorLogPercent),
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
		Prefix:            max(a.Source, b.Source),
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

var combineTagSetLists = combineSets[map[string][]string](func(a, b map[string][]string) bool {
	return reflect.DeepEqual(a, b)
})

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

func combineStringSet(a, b []string) []string {
	var c []string
	if len(a) > 0 {
		c = make([]string, len(a))
		copy(c, a)
	}
	sort.Strings(c)
	for _, tag := range b {
		i := sort.SearchStrings(c, tag)
		if i < len(c) && c[i] == tag {
			continue
		}
		c = append(c, tag)
		sort.Strings(c)
	}
	return c
}

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
