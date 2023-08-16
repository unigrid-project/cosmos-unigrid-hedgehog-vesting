package ugdvesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "ugdvesting/testutil/keeper"
	"ugdvesting/testutil/nullify"
	"ugdvesting/x/ugdvesting"
	"ugdvesting/x/ugdvesting/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.UgdvestingKeeper(t)
	ugdvesting.InitGenesis(ctx, *k, genesisState)
	got := ugdvesting.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
