package plugin

import "testing"

func TestPipelineExecuteHandledAbortsChain(t *testing.T) {
	t.Parallel()

	calls := 0
	p := NewPipeline([]PipelinePlugin{
		{
			name:     "first",
			phase:    PhasePreProxy,
			priority: 1,
			run: func(ctx *PipelineContext) (bool, error) {
				calls++
				return true, nil
			},
		},
		{
			name:     "second",
			phase:    PhasePreProxy,
			priority: 2,
			run: func(ctx *PipelineContext) (bool, error) {
				calls++
				return false, nil
			},
		},
	})

	ctx := &PipelineContext{}
	handled, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true")
	}
	if calls != 1 {
		t.Fatalf("expected one plugin call, got %d", calls)
	}
	if !ctx.Aborted {
		t.Fatalf("expected context to be aborted")
	}
}

func TestPipelineExecuteAbortFlagStopsExecution(t *testing.T) {
	t.Parallel()

	calls := 0
	p := NewPipeline([]PipelinePlugin{
		{
			name:     "first",
			phase:    PhasePreProxy,
			priority: 1,
			run: func(ctx *PipelineContext) (bool, error) {
				calls++
				ctx.Abort("blocked")
				return false, nil
			},
		},
		{
			name:     "second",
			phase:    PhasePreProxy,
			priority: 2,
			run: func(ctx *PipelineContext) (bool, error) {
				calls++
				return false, nil
			},
		},
	})

	ctx := &PipelineContext{}
	handled, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true when context aborts")
	}
	if calls != 1 {
		t.Fatalf("expected one plugin call after abort, got %d", calls)
	}
	if ctx.AbortReason != "blocked" {
		t.Fatalf("unexpected abort reason %q", ctx.AbortReason)
	}
}

func TestPipelineExecutePostProxyRunsAfterCallbacks(t *testing.T) {
	t.Parallel()

	calls := 0
	p := NewPipeline([]PipelinePlugin{
		{
			name:     "first",
			phase:    PhasePreProxy,
			priority: 1,
		},
		{
			name:     "second",
			phase:    PhasePostProxy,
			priority: 40,
			after: func(ctx *PipelineContext, proxyErr error) {
				calls++
			},
		},
	})

	p.ExecutePostProxy(&PipelineContext{}, nil)
	if calls != 1 {
		t.Fatalf("expected one after-proxy callback call, got %d", calls)
	}
}
