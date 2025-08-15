package bank

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customcli "github.com/classic-terra/core/v3/custom/bank/client/cli"
	customsim "github.com/classic-terra/core/v3/custom/bank/simulation"
	customtypes "github.com/classic-terra/core/v3/custom/bank/types"
	customkeeper "github.com/efeovez/columbus-core/custom/bank/keeper"
)

var (
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModule           = AppModule{}
	_ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the distribution module.
type AppModuleBasic struct {
	bank.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the bank module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
	*types.ModuleCdc = *customtypes.ModuleCdc
}

// GetTxCmd returns the root tx command for the bank module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return customcli.GetTxCmd()
}

// AppModule implements an application module for the bank module.
// AppModule wraps around the bank module and the bank keeper to return the right total supply ignoring bonded tokens
// that the alliance module minted to rebalance the voting power
// It modifies the TotalSupply and SupplyOf GRPC queries
type AppModule struct {
	bank.AppModule
	keeper        customkeeper.Keeper
	accountKeeper types.AccountKeeper
	subspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper customkeeper.Keeper, accountKeeper types.AccountKeeper, ss exported.Subspace) AppModule {
	return AppModule{
		AppModule:     bank.NewAppModule(cdc, keeper, accountKeeper, ss),
		keeper:        keeper,
		accountKeeper: accountKeeper,
		subspace:      ss,
	}
}

// GenerateGenesisState creates a randomized GenState of the bank module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	customsim.RandomizedGenState(simState)
}

// WeightedOperations return random bank module operation.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	// We implement our own bank operations to prevent insufficient fee due to tax
	return customsim.WeightedOperations(
		simState.AppParams, simState.Cdc, am.accountKeeper, am.keeper,
	)
}

// RegisterServices registers module services.
// NOTE: Overriding this method as not doing so will cause a panic
// when trying to force this custom keeper into a bankkeeper.BaseKeeper
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	m := keeper.NewMigrator(am.keeper.BaseKeeper, am.subspace)
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 1 to 2: %v", err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 2 to 3: %v", err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 3 to 4: %v", err))
	}
}