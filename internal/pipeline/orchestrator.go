package pipeline

import "errors"

// OrchestratorOption configures the orchestrator.
type OrchestratorOption func(*Orchestrator)

type Orchestrator struct {
	runner   Runner
	policy   RollbackPolicy
	stepByID map[string]Step
}

func NewOrchestrator(policy RollbackPolicy, opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		runner:   Runner{},
		policy:   policy,
		stepByID: map[string]Step{},
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

func (o *Orchestrator) Execute(plan StagePlan) ExecutionResult {
	o.indexSteps(plan.Prepare)
	o.indexSteps(plan.Apply)

	prepareResult := o.runner.Run(StagePrepare, plan.Prepare)
	if !prepareResult.Success {
		return ExecutionResult{Prepare: prepareResult, Err: prepareResult.Err}
	}

	applyResult := o.runner.Run(StageApply, plan.Apply)
	result := ExecutionResult{Prepare: prepareResult, Apply: applyResult}
	if applyResult.Success {
		return result
	}

	result.Err = applyResult.Err
	if o.policy.ShouldRollback(StageApply, applyResult.Err) {
		result.Rollback = ExecuteRollback(applyResult.Steps, o.stepByID)
		if !result.Rollback.Success {
			result.Err = errors.Join(result.Err, result.Rollback.Err)
		}
	}

	return result
}

// Rollback compensates successful apply steps after a downstream consumer,
// such as state persistence, fails. It must not be called after apply failure,
// because Execute already performs that rollback.
func (o *Orchestrator) Rollback(result ExecutionResult) StageResult {
	return ExecuteRollback(result.Apply.Steps, o.stepByID)
}

func (o *Orchestrator) indexSteps(steps []Step) {
	for _, step := range steps {
		o.stepByID[step.ID()] = step
	}
}
