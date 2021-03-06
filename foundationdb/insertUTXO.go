package foundationdb

import (
	"bytes"
	"errors"

	fdb "github.com/apple/foundationdb/bindings/go/src/fdb"
	commonConst "github.com/bankex/go-plasma/common"
	transaction "github.com/bankex/go-plasma/transaction"
)

type UTXOinserter struct {
	db *fdb.Database
}

func NewUTXOinserter(db *fdb.Database) *UTXOinserter {
	reader := &UTXOinserter{db: db}
	return reader
}

func (r *UTXOinserter) InsertUTXO(tx *transaction.SignedTransaction, blockNumber uint32, transactionNumber uint32) error {
	numOutputs := len(tx.UnsignedTransaction.Outputs)
	utxoIndexes := make([][]byte, numOutputs)
	for i := 0; i < numOutputs; i++ {
		utxoIndex, err := transaction.CreateUTXOIndexForOutput(tx, blockNumber, transactionNumber, i)
		if err != nil {
			return err
		}
		fullIndex := []byte{}
		fullIndex = append(fullIndex, commonConst.UtxoIndexPrefix...)
		fullIndex = append(fullIndex, utxoIndex[:]...)
		utxoIndexes[i] = fullIndex
	}

	ret, err := r.db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		for _, index := range utxoIndexes {
			existing, err := tr.Get(fdb.Key(index)).Get()
			if err != nil || len(existing) != 0 {
				return nil, err
			}
		}
		for _, index := range utxoIndexes {
			tr.Set(fdb.Key(index), []byte{commonConst.UTXOisReadyForSpending})
		}
		for _, index := range utxoIndexes {
			existing, err := tr.Get(fdb.Key(index)).Get()
			if err != nil {
				tr.Reset()
				return nil, err
			}
			if len(existing) != 1 || bytes.Compare(existing, []byte{commonConst.UTXOisReadyForSpending}) != 0 {
				tr.Reset()
				return nil, errors.New("Reading mismatch")
			}
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	if ret == nil {
		return errors.New("Could not write a transaction")
	}
	return nil
}
