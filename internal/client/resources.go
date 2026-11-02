package client

import (
	"context"
	"fmt"
	"net/http"
)

// ListMachines returns all machines in a project.
func (c *Client) ListMachines(ctx context.Context, projectID string) ([]Machine, error) {
	return listAll[Machine](ctx, c, projectPath(projectID, "machines"))
}

// GetMachine returns a single machine.
func (c *Client) GetMachine(ctx context.Context, projectID, machineID string) (*Machine, error) {
	var m Machine
	if err := c.do(ctx, http.MethodGet, projectPath(projectID, "machines", machineID), nil, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListBlueprints returns all blueprints in a project.
func (c *Client) ListBlueprints(ctx context.Context, projectID string) ([]Blueprint, error) {
	return listAll[Blueprint](ctx, c, projectPath(projectID, "blueprints"))
}

// GetBlueprint returns a single blueprint.
func (c *Client) GetBlueprint(ctx context.Context, projectID, blueprintID string) (*Blueprint, error) {
	var bp Blueprint
	if err := c.do(ctx, http.MethodGet, projectPath(projectID, "blueprints", blueprintID), nil, nil, &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}

// FindBlueprintByMachineID returns the blueprint attached to a machine, or
// nil (and no error) if the machine has no blueprint.
func (c *Client) FindBlueprintByMachineID(ctx context.Context, projectID, machineID string) (*Blueprint, error) {
	blueprints, err := c.ListBlueprints(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range blueprints {
		if blueprints[i].MachineID == machineID {
			return &blueprints[i], nil
		}
	}
	return nil, nil
}

// CreateBlueprint creates a blueprint. bp.MachineID is required.
func (c *Client) CreateBlueprint(ctx context.Context, projectID string, bp *Blueprint) (*Blueprint, error) {
	if bp.MachineID == "" {
		return nil, fmt.Errorf("cloudendure: machineId is required to create a blueprint")
	}
	var out Blueprint
	if err := c.do(ctx, http.MethodPost, projectPath(projectID, "blueprints"), nil, bp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBlueprint patches an existing blueprint and returns the updated object.
func (c *Client) UpdateBlueprint(ctx context.Context, projectID, blueprintID string, bp *Blueprint) (*Blueprint, error) {
	var out Blueprint
	if err := c.do(ctx, http.MethodPatch, projectPath(projectID, "blueprints", blueprintID), nil, bp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRecoveryPlans returns all recovery plans in a project.
func (c *Client) ListRecoveryPlans(ctx context.Context, projectID string) ([]RecoveryPlan, error) {
	return listAll[RecoveryPlan](ctx, c, projectPath(projectID, "recoveryPlans"))
}

// GetRecoveryPlan returns a single recovery plan.
func (c *Client) GetRecoveryPlan(ctx context.Context, projectID, planID string) (*RecoveryPlan, error) {
	var rp RecoveryPlan
	if err := c.do(ctx, http.MethodGet, projectPath(projectID, "recoveryPlans", planID), nil, nil, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}

// CreateRecoveryPlan creates a recovery plan.
func (c *Client) CreateRecoveryPlan(ctx context.Context, projectID string, rp *RecoveryPlan) (*RecoveryPlan, error) {
	var out RecoveryPlan
	if err := c.do(ctx, http.MethodPost, projectPath(projectID, "recoveryPlans"), nil, rp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRecoveryPlan patches a recovery plan.
func (c *Client) UpdateRecoveryPlan(ctx context.Context, projectID, planID string, rp *RecoveryPlan) (*RecoveryPlan, error) {
	var out RecoveryPlan
	if err := c.do(ctx, http.MethodPatch, projectPath(projectID, "recoveryPlans", planID), nil, rp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteRecoveryPlan deletes a recovery plan.
func (c *Client) DeleteRecoveryPlan(ctx context.Context, projectID, planID string) error {
	return c.do(ctx, http.MethodDelete, projectPath(projectID, "recoveryPlans", planID), nil, nil, nil)
}
