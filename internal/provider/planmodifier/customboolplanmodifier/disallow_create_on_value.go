package customboolplanmodifier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func DisallowOnCreateOnValue(value bool) planmodifier.Bool {
	return disallowOnCreateOnValueModifier{
		value: value,
	}
}

// disallowOnCreateOnValueModifier is an plan modifier that sets RequiresReplace
// on the attribute if a given function is true.
type disallowOnCreateOnValueModifier struct {
	value bool
}

// Description returns a human-readable description of the plan modifier.
func (m disallowOnCreateOnValueModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Cannot create resource with attribute set to %t", m.value)
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m disallowOnCreateOnValueModifier) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("Cannot create resource with attribute set to %t", m.value)
}

// PlanModifyBool implements the plan modification logic.
func (m disallowOnCreateOnValueModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() && req.PlanValue.Equal(types.BoolValue(m.value)) {
		resp.Diagnostics.AddAttributeError(req.Path, "Client Error", fmt.Sprintf("Cannot create resource with attribute set to %t", m.value))
	}
}
