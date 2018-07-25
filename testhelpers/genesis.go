package testhelpers

import (
	"context"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/actor/builtin/account"
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/actor/builtin/paymentbroker"
	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"

	"gx/ipfs/QmcYBp5EDnJKfVN63F71rDTksvEf1cfijwCTWtw6bPG58T/go-hamt-ipld"
)

// Config is used to configure values in the GenesisInitFunction
type Config struct {
	accounts map[types.Address]*types.AttoFIL
	nonces   map[types.Address]uint64
	miners   map[types.Address]*types.BytesAmount
}

// GenOption is a configuration option for the GenesisInitFunction
type GenOption func(*Config) error

// ActorAccount returns a config option that sets up an actor account
func ActorAccount(addr types.Address, amt *types.AttoFIL) GenOption {
	return func(gc *Config) error {
		gc.accounts[addr] = amt
		return nil
	}
}

// ActorNonce returns a config option that sets the nonce of an existing actor
func ActorNonce(addr types.Address, nonce uint64) GenOption {
	return func(gc *Config) error {
		gc.nonces[addr] = nonce
		return nil
	}
}

// MinerPower returns a config option that sets up a miner actor with a
// network power.
func MinerPower(addr types.Address, pwr *types.BytesAmount) GenOption {
	return func(gc *Config) error {
		gc.miners[addr] = pwr
		return nil
	}
}

// NewEmptyConfig inits and returns an empty config
func NewEmptyConfig() *Config {
	genCfg := &Config{}
	genCfg.accounts = make(map[types.Address]*types.AttoFIL)
	genCfg.nonces = make(map[types.Address]uint64)
	genCfg.miners = make(map[types.Address]*types.BytesAmount)
	return genCfg
}

// MakeGenesisFunc is a method used to define a custom genesis function
func MakeGenesisFunc(opts ...GenOption) func(cst *hamt.CborIpldStore) (*types.Block, error) {
	gif := func(cst *hamt.CborIpldStore) (*types.Block, error) {
		genCfg := NewEmptyConfig()
		for _, opt := range opts {
			opt(genCfg) // nolint: errcheck
		}

		ctx := context.Background()
		st := state.NewEmptyStateTree(cst)

		// Initialize account actors
		for addr, val := range genCfg.accounts {
			a, err := account.NewActor(val)
			if err != nil {
				return nil, err
			}

			if err := st.SetActor(ctx, addr, a); err != nil {
				return nil, err
			}
		}
		for addr, nonce := range genCfg.nonces {
			a, err := st.GetActor(ctx, addr)
			if err != nil {
				return nil, err
			}
			a.Nonce = types.Uint64(nonce)
			if err := st.SetActor(ctx, addr, a); err != nil {
				return nil, err
			}
		}

		// Create NetworkAddress
		a, err := account.NewActor(types.NewAttoFILFromFIL(10000000))
		if err != nil {
			return nil, err
		}
		if err := st.SetActor(ctx, address.NetworkAddress, a); err != nil {
			return nil, err
		}

		// Initialize miner actors
		for addr, pwr := range genCfg.miners {
			mstore := &miner.Storage{
				Owner:       addr,
				Power:       pwr,
				PledgeBytes: pwr,
			}
			storageBytes, err := actor.MarshalStorage(mstore)
			if err != nil {
				return nil, err
			}
			a := types.NewActorWithMemory(types.MinerActorCodeCid, nil, storageBytes)
			if err := st.SetActor(ctx, addr, a); err != nil {
				return nil, err
			}
		}

		// Initialize storage market actor
		stAct, err := storagemarket.NewActor()
		if err != nil {
			return nil, err
		}
		var sstore storagemarket.Storage
		if len(genCfg.miners) > 0 {
			err = actor.UnmarshalStorage(stAct.Memory, &sstore)
			if err != nil {
				return nil, err
			}
			for addr, pwr := range genCfg.miners {
				sstore.Miners[addr] = struct{}{}
				sstore.TotalCommittedStorage = sstore.TotalCommittedStorage.Add(pwr)
			}
			sstoreBytes, err := actor.MarshalStorage(sstore)
			if err != nil {
				return nil, err
			}
			stAct = types.NewActorWithMemory(types.StorageMarketActorCodeCid, nil, sstoreBytes)
		}

		if err := st.SetActor(ctx, address.StorageMarketAddress, stAct); err != nil {
			return nil, err
		}

		// Create PaymentBrokerAddress
		pbAct, err := paymentbroker.NewPaymentBrokerActor()
		pbAct.Balance = types.NewAttoFILFromFIL(0)
		if err != nil {
			return nil, err
		}
		if err := st.SetActor(ctx, address.PaymentBrokerAddress, pbAct); err != nil {
			return nil, err
		}

		c, err := st.Flush(ctx)
		if err != nil {
			return nil, err
		}

		genesis := &types.Block{
			StateRoot: c,
			Nonce:     1337,
		}

		if _, err := cst.Put(ctx, genesis); err != nil {
			return nil, err
		}

		return genesis, nil
	}

	// Pronounced "Jif" - JenesisInitFunction
	return gif
}
