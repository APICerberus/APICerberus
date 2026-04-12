package plugin

// Pipeline contains an ordered plugin chain for one route.
type Pipeline struct {
	plugins []PipelinePlugin
}

func NewPipeline(plugins []PipelinePlugin) *Pipeline {
	cloned := make([]PipelinePlugin, len(plugins))
	copy(cloned, plugins)
	return &Pipeline{plugins: cloned}
}

// Execute runs pre-proxy phases and can abort early.
func (p *Pipeline) Execute(ctx *PipelineContext) (bool, error) {
	if p == nil || ctx == nil {
		return false, nil
	}

	for _, plugin := range p.plugins {
		handled, err := plugin.Run(ctx)
		if err != nil {
			return false, err
		}
		if handled {
			if !ctx.Aborted {
				ctx.Abort(plugin.name + ": handled response")
			}
			return true, nil
		}
		if ctx.Aborted {
			return true, nil
		}
	}
	return false, nil
}

// ExecutePostProxy runs post-proxy callbacks (including retries/circuit hooks).
func (p *Pipeline) ExecutePostProxy(ctx *PipelineContext, proxyErr error) {
	if p == nil || ctx == nil {
		return
	}
	for _, plugin := range p.plugins {
		plugin.AfterProxy(ctx, proxyErr)
	}
}

func (p *Pipeline) Plugins() []PipelinePlugin {
	if p == nil {
		return nil
	}
	out := make([]PipelinePlugin, len(p.plugins))
	copy(out, p.plugins)
	return out
}
