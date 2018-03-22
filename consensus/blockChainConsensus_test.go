package consensus

import (
    "testing"
)

func TestNewBlockChainConsensus(t *testing.T) {
	// verify that blockchain instance implements Consensus interface
	var consensus Consensus
	consensus, err := NewBlockChainConsensus()
	if err != nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if consensus == nil {
		t.Errorf("got nil instance")
		return
	}
}

func TestMineCandidateBlock(t *testing.T) {
	consensus, err := NewBlockChainConsensus()
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if err = consensus.MineCandidateBlock(nil); err != nil {
		t.Errorf("failed to mine candidate block: %s", err)
		return
	}
}

func TestTransactionStatus(t *testing.T) {
	consensus, err := NewBlockChainConsensus()
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	var b Block
	if b,err = consensus.TransactionStatus(nil); err != nil {
		t.Errorf("failed to get transaction status: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
}

func TestDeserialize(t *testing.T) {
	consensus, err := NewBlockChainConsensus()
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	var b Block
	if b, err = consensus.Deserialize(nil); err != nil {
		t.Errorf("failed to deserialize block: %s", err)
		return
	}
	if b == nil {
		t.Errorf("got nil instance")
		return
	}
}

func TestAcceptNetworkBlock(t *testing.T) {
	consensus, err := NewBlockChainConsensus()
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	if err = consensus.AcceptNetworkBlock(nil); err != nil {
		t.Errorf("failed to get accept network block: %s", err)
		return
	}
}

func TestSerialize(t *testing.T) {
	consensus, err := NewBlockChainConsensus()
	if err != nil || consensus == nil {
		t.Errorf("failed to get blockchain consensus instance: %s", err)
		return
	}
	var data []byte
	if data, err = consensus.Serialize(nil); err != nil {
		t.Errorf("failed to srialize block: %s", err)
		return
	}
	if len(data) == 0 {
		t.Errorf("got empty serialized data")
		return
	}
}

