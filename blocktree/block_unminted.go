package blocktree

import (
	"errors"
)

// Unminted block struct
type UnmintedBlock struct {
	Transactions map[string]Transaction
	Previous     Hash
}

// Create new block data
func NewBlock(previous Hash) UnmintedBlock {
	return UnmintedBlock{map[string]Transaction{}, previous}
}

// Add transaction
func (b *UnmintedBlock) AddTx(tree Blocktree, from string, outputs ...TxOutput) error {
	var inputs []TxInput

	// Calculate total amount
	amount := uint64(0)
	for _, r := range outputs {
		amount += r.Value
	}

	// Check if there are funds to perform transaction
	acc, validOutputs := tree.findSpendableOutputs(from, amount, b.Previous)
	if acc < amount {
		return errors.New("not enough funds")
	}

	// Generate inputs and outputs
	for tx, outs := range validOutputs {
		for _, out := range outs {
			input := TxInput{tx, out}
			inputs = append(inputs, input)
		}
	}

	// Calculate change
	if acc > amount {
		outputs = append(outputs, TxOutput{from, acc - amount})
	}

	// Create and append transactions
	tx := Transaction{Inputs: inputs, Outputs: outputs}
	b.Transactions[from] = tx
	return nil
}

// Add reward transaction
func (b *UnmintedBlock) AddRewardTx(tree Blocktree, to string) {
	// Coinbase input
	input := TxInput{NilHash, 0}
	// Miner output
	output := TxOutput{to, tree.network.Reward}
	// Transaction
	tx := Transaction{Inputs: []TxInput{input}, Outputs: []TxOutput{output}}
	// Append transaction
	b.Transactions["coinbase"] = tx
}