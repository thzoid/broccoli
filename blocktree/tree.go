package blocktree

import (
	"errors"
	"fmt"

	"github.com/thzoid/broccoli/hash"
	"github.com/thzoid/broccoli/wallet"
)

// Blocktree struct
// This structure exists only to ensure
// the integrity of the tree (i.e. not
// allowing other functions to modify
// the actual blocks)
type Blocktree struct {
	blocks  map[hash.Hash]MintedBlock
	network Network
}

// Create new tree
func NewTree(n Network, miner wallet.Address) (bt Blocktree, root hash.Hash) {
	bt = Blocktree{map[hash.Hash]MintedBlock{}, n}
	// Create root block
	root = bt.Graft(NewBlock(hash.NilHash), miner)
	return
}

// Perform validations and return expected minted block to be grafted
func (bt *Blocktree) Mint(ub UnmintedBlock, miner wallet.Address) (*MintedBlock, error) {
	// Check if block is root but there are blocks on the tree already
	if ub.Previous == hash.NilHash {
		if len(bt.blocks) != 0 {
			return nil, errors.New("multiple root blocks are not allowed")
		}
		// Check if block exists
	} else if bt.Block(ub.Previous) == nil {
		return nil, errors.New("previous block not found")
	}
	// Check transactions
	for _, t := range ub.Transactions.Keys() {
		if val, _ := ub.Transactions.Get(t); val.(Transaction).IsFromCoinbase() {
			return nil, errors.New("illegal coinbase transaction found")
		}
	}
	// Add reward from coinbase to miner
	ub.AddRewardTx(*bt, miner)

	return &MintedBlock{
		tree: bt,

		transactions: ub.Transactions,
		previous:     ub.Previous,
		nonce:        0,
	}, nil
}

// Graft (mine and add) a block into a branch
func (bt *Blocktree) Graft(ub UnmintedBlock, miner wallet.Address) hash.Hash {
	// Get minted block
	b, err := bt.Mint(ub, miner)
	if err != nil {
		panic(err)
	}
	// Mine block
	b.mine(bt.network)
	hash := b.Hash()
	// Add to blocktree
	bt.blocks[hash] = *b
	return hash
}

// View tree
func (bt *Blocktree) View(branch hash.Hash) {
	for i, b := 0, branch; b != hash.NilHash; i, b = i+1, bt.blocks[b].previous {
		if bt.blocks[b].previous == hash.NilHash {
			fmt.Printf("╳  #r\t%x\n", b)
		} else if i == 0 {
			fmt.Printf("┌─ #%d\t%x\n", i, b)
		} else {
			fmt.Printf("├─ #%d\t%x\n", i, b)
		}
	}
}

// Get list of unspent transactions
func (bt *Blocktree) findUnspentTxs(address wallet.Address, branch hash.Hash) []Transaction {
	var unspentTxs []Transaction
	spentTxIDs := map[hash.Hash][]uint8{}

	for b := branch; b != hash.NilHash; b = bt.blocks[b].previous {
		txs := bt.blocks[b].transactions
		for el := txs.Front(); el != nil; el = el.Next() {
			tx := el.Value.(Transaction)
			txHash := tx.Hash()
		Outputs:
			for i, out := range tx.Outputs {
				if spentTxIDs[txHash] != nil {
					for _, spentOut := range spentTxIDs[txHash] {
						if spentOut == uint8(i) {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, tx)
				}
			}
			if !tx.IsFromCoinbase() {
				for _, in := range tx.Inputs {
					inTxID := in.ID
					spentTxIDs[txHash] = append(spentTxIDs[inTxID], in.Index)
				}
			}
		}
	}

	return unspentTxs
}

// Get spendable outputs for the provided wallet
func (tree *Blocktree) findSpendableOutputs(address wallet.Address, amount uint64, branch hash.Hash) (uint64, map[hash.Hash][]uint8) {
	unspentOuts := map[hash.Hash][]uint8{}
	unspentTxs := tree.findUnspentTxs(address, branch)
	accumulated := uint64(0)

Work:
	for _, tx := range unspentTxs {
		hash := tx.Hash()
		for i, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[hash] = append(unspentOuts[hash], uint8(i))

				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOuts
}

// Find a block in the Blocktree
// Returns a pointer to the block (not the actual reference
// to the minted block) if found or nil otherwise
func (bt *Blocktree) Block(hash hash.Hash) *MintedBlock {
	b, ok := bt.blocks[hash]
	if !ok {
		return nil
	} else {
		return &b
	}
}
