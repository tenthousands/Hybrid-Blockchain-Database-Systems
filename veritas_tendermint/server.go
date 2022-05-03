package veritastm

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	pbv "github.com/nusdbsystem/hybridproto/veritas"
)

type server struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *Config

	Ledger *LedgerApp

	redisCli *redis.Client
	abciCli  *rpchttp.HTTP

	signature string
}

type BlockPurpose struct {
	blk      *pbv.Block
	approved map[string]struct{}
}

func NewServer(rcli *redis.Client, config *Config) *server {
	ctx, cancel := context.WithCancel(context.Background())

	// ledger app
	lapp := NewLedgerApp(config, rcli)

	// ABCI Client
	var abciClient *rpchttp.HTTP
	// abciClient = nil
	abciClient, err := rpchttp.New(config.ABCIRPCAddr)
	if err != nil || abciClient == nil {
		fmt.Printf("Error setting ABCI client: %v\n", err)
	}

	s := &server{
		ctx:       ctx,
		cancel:    cancel,
		Ledger:    lapp,
		abciCli:   abciClient,
		config:    config,
		redisCli:  rcli,
		signature: config.Signature,
	}
	return s
}

func (s *server) Get(ctx context.Context, req *pbv.GetRequest) (*pbv.GetResponse, error) {
	res, err := s.redisCli.Get(ctx, req.GetKey()).Result()
	if err != nil {
		return nil, err
	}

	return &pbv.GetResponse{Value: res}, nil
}

func (s *server) Set(ctx context.Context, req *pbv.SetRequest) (*pbv.SetResponse, error) {
	// check version
	getReq := &pbv.GetRequest{
		Signature: req.GetSignature(),
		Key:       req.GetKey(),
	}
	record, _ := s.Get(ctx, getReq)
	if record != nil && record.Version > req.GetVersion() {
		return &pbv.SetResponse{}, errors.New("Rejected (wrong version)")
	}
	// send transaction to ledger and wait for it to be committed
	// Format: key=value#version#signature+timestamp
	// append node signature and timestamp to make it unique
	t := time.Now().Unix()
	tx := fmt.Sprintf("%s=%s#%d#%s%d", req.GetKey(), req.GetValue(), req.GetVersion(), s.signature, t)

	res, err := s.abciCli.BroadcastTxCommit(s.ctx, []byte(tx))
	if err != nil {
		fmt.Printf("Error in Set: %v\n", err)
		return nil, err
	}

	if res.CheckTx.IsErr() || res.DeliverTx.IsErr() {
		fmt.Println("Error in Set: BroadcastTxCommit transaction failed")
		return nil, errors.Wrap(err, "BroadcastTxCommit transaction failed")
	}

	return &pbv.SetResponse{
		Txid: res.Hash.String(),
	}, nil
}

func (s *server) Verify(ctx context.Context, req *pbv.VerifyRequest) (*pbv.VerifyResponse, error) {
	/*
		proof, err := s.l.ProveKey([]byte(req.GetKey()))
		if err != nil {
			return nil, err
		}
		return &pbv.VerifyResponse{
			RootDigest:            s.l.GetRootDigest(),
			SideNodes:             proof.SideNodes,
			NonMembershipLeafData: proof.NonMembershipLeafData,
		}, nil
	*/
	return nil, nil
}
