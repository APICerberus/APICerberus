package config

// CloneConfig returns a deep copy of the runtime configuration snapshot.
func CloneConfig(src *Config) *Config {
	if src == nil {
		return &Config{}
	}
	out := *src
	if len(src.Audit.RouteRetentionDays) > 0 {
		out.Audit.RouteRetentionDays = make(map[string]int, len(src.Audit.RouteRetentionDays))
		for route, days := range src.Audit.RouteRetentionDays {
			out.Audit.RouteRetentionDays[route] = days
		}
	}
	out.Billing = CloneBillingConfig(src.Billing)
	out.Services = append([]Service(nil), src.Services...)
	out.Routes = append([]Route(nil), src.Routes...)
	out.GlobalPlugins = ClonePluginConfigs(src.GlobalPlugins)
	for i := range out.Routes {
		out.Routes[i].Plugins = ClonePluginConfigs(src.Routes[i].Plugins)
	}

	out.Upstreams = append([]Upstream(nil), src.Upstreams...)
	for i := range out.Upstreams {
		out.Upstreams[i].Targets = append([]UpstreamTarget(nil), src.Upstreams[i].Targets...)
	}

	out.Consumers = append([]Consumer(nil), src.Consumers...)
	for i := range out.Consumers {
		out.Consumers[i].APIKeys = append([]ConsumerAPIKey(nil), src.Consumers[i].APIKeys...)
		out.Consumers[i].ACLGroups = append([]string(nil), src.Consumers[i].ACLGroups...)
		if src.Consumers[i].Metadata != nil {
			out.Consumers[i].Metadata = make(map[string]any, len(src.Consumers[i].Metadata))
			for k, v := range src.Consumers[i].Metadata {
				out.Consumers[i].Metadata[k] = v
			}
		}
	}
	out.Auth.APIKey.KeyNames = append([]string(nil), src.Auth.APIKey.KeyNames...)
	out.Auth.APIKey.QueryNames = append([]string(nil), src.Auth.APIKey.QueryNames...)
	out.Auth.APIKey.CookieNames = append([]string(nil), src.Auth.APIKey.CookieNames...)
	return &out
}

// ClonePluginConfigs returns a deep copy of a plugin config slice.
// The *bool Enabled pointer and map[string]any Config are deep-copied.
func ClonePluginConfigs(in []PluginConfig) []PluginConfig {
	if len(in) == 0 {
		return nil
	}
	out := make([]PluginConfig, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Enabled != nil {
			v := *in[i].Enabled
			out[i].Enabled = &v
		}
		if in[i].Config != nil {
			out[i].Config = make(map[string]any, len(in[i].Config))
			for k, v := range in[i].Config {
				out[i].Config[k] = v
			}
		}
	}
	return out
}

// CloneBillingConfig returns a deep copy of the billing configuration.
func CloneBillingConfig(in BillingConfig) BillingConfig {
	out := in
	out.RouteCosts = CloneInt64Map(in.RouteCosts)
	out.MethodMultipliers = CloneFloat64Map(in.MethodMultipliers)
	return out
}

// CloneInt64Map returns a shallow copy of a map[string]int64.
// Returns an empty non-nil map for zero-length input.
func CloneInt64Map(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// CloneFloat64Map returns a shallow copy of a map[string]float64.
// Returns an empty non-nil map for zero-length input.
func CloneFloat64Map(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// CloneAnyMap returns a shallow copy of a map[string]any.
// Returns an empty non-nil map for zero-length input.
func CloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
