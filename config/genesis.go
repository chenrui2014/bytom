package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

// Generate genesis transaction
func GenerateGenesisTx() *legacy.Tx {
	txData := legacy.TxData{
		Version:        1,
		SerializedSize: 63,
		Inputs:         []*legacy.TxInput{},
		Outputs: []*legacy.TxOutput{
			&legacy.TxOutput{
				AssetVersion: 1,
				OutputCommitment: legacy.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetId: consensus.BTMAssetID,
						Amount:  1470000000000000000,
					},
					VMVersion:      1,
					ControlProgram: []byte{81},
				},
			},
		},
		MinTime: 0,
		MaxTime: 1511318565142,
	}

	return legacy.NewTx(txData)
}

// Generate genesis block
func GenerateGenesisBlock() *legacy.Block {
	genesisCoinbaseTx := GenerateGenesisTx()
	merkleRoot, err := bc.MerkleRoot([]*bc.Tx{genesisCoinbaseTx.Tx})
	if err != nil {
		log.Panicf("Fatal create merkelRoot")
	}

	snap := state.Empty()
	if err := snap.ApplyTx(genesisCoinbaseTx.Tx); err != nil {
		log.Panicf("Fatal ApplyTx")
	}

	genesisBlock := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:     1,
			Height:      1,
			Seed:        bc.NewHash([32]byte{}),
			TimestampMS: 1511318565142,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				AssetsMerkleRoot:       snap.Tree.RootHash(),
			},
			Bits: 2161727821138738707,
		},
		Transactions: []*legacy.Tx{genesisCoinbaseTx},
	}

	for i := uint64(0); i <= 10000000000000; i++ {
		genesisBlock.Nonce = i
		hash := genesisBlock.Hash()

		if consensus.CheckProofOfWork(&hash, genesisBlock.Bits) {
			break
		}
	}

	log.Infof("genesisBlock:%v", genesisBlock)
	return genesisBlock
}