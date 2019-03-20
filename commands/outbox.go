package commands

import (
	"io"

	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
	"gx/ipfs/Qmf46mr235gtyxizkKUkTH5fo62Thza2zwXR4DWC7rkoqF/go-ipfs-cmds"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/core"
)

var outboxCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "View and manipulate the outbound message queue",
	},
	Subcommands: map[string]*cmds.Command{
		"ls":    outboxLsCmd,
		"clear": outboxClearCmd,
	},
}

// OutboxLsResult is a listing of the outbox for a single address.
type OutboxLsResult struct {
	Address  address.Address
	Messages []*core.QueuedMessage
}

var outboxLsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List the queue(s) of sent but un-mined messages",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", false, false, "Address the queue to list (otherwise lists all)"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addresses, err := queueAddressesFromArg(req, env, 0)
		if err != nil {
			return err
		}

		for _, addr := range addresses {
			msgs := GetPorcelainAPI(env).OutboxQueueLs(addr)
			err := re.Emit(OutboxLsResult{addr, msgs})
			if err != nil {
				return err
			}
		}
		return nil
	},
	Type: OutboxLsResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, queue *OutboxLsResult) error {
			sw := NewSilentWriter(w)
			sw.Println("From:", queue.Address.String())
			for _, qm := range queue.Messages {
				msg := qm.Msg
				cid, err := msg.Cid()
				if err != nil {
					return err
				}
				sw.Printf("CID: %s, nonce: %d, gas limit: %d, height: %d\n", cid.String(), msg.Nonce, msg.GasLimit, qm.Stamp)
			}
			return sw.Error()
		}),
	},
}

var outboxClearCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Clear the queue(s) of sent messages",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", false, false, "Address the queue to clear (otherwise clears all)"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addresses, err := queueAddressesFromArg(req, env, 0)
		if err != nil {
			return err
		}

		for _, addr := range addresses {
			GetPorcelainAPI(env).OutboxQueueClear(addr)
		}
		return nil
	},
	Encoders: cmds.EncoderMap{},
}

// Reads an address from an argument, or lists addresses of all outbox queues if no arg is given.
func queueAddressesFromArg(req *cmds.Request, env cmds.Environment, argIndex int) ([]address.Address, error) {
	var addresses []address.Address
	if len(req.Arguments) > argIndex {
		addr, e := address.NewFromString(req.Arguments[argIndex])
		if e != nil {
			return nil, e
		}
		addresses = []address.Address{addr}
	} else {
		addresses = GetPorcelainAPI(env).OutboxQueues()
	}
	return addresses, nil
}
