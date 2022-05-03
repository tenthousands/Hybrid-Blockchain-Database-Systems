package sharding

import (
	"context"
	"hash/adler32"
	"log"
	"strconv"
	"strings"

	"github.com/nusdbsystem/hybridBlockchainDB/bcdbnode/config"
	Connectors "github.com/nusdbsystem/hybridBlockchainDB/blockchainconnectors"
	EthClientSDK "github.com/nusdbsystem/hybridBlockchainDB/storage/ethereum/clientSDK"
	//FabClientSDK "github.com/nusdbsystem/hybridBlockchainDB/storage/fabric/clientSDK"
)

type ShardingMgr struct {
	LocalShard  Connectors.BlockchainConnector
	Shards      map[string]Connectors.BlockchainConnector
	Conf        map[string]config.Shard
	ShardNumber int
}

func NewShardingMgr(conf *config.Options) (*ShardingMgr, error) {
	var localconn Connectors.BlockchainConnector
	shards := make(map[string]Connectors.BlockchainConnector)
	confs := make(map[string]config.Shard)
	for _, shard := range conf.Shards {
		switch shard.Type {
		case PARTITION_ETH().Shard:
			if strings.HasPrefix(conf.SelfID, shard.ID) {
				ethconn, err := EthClientSDK.NewEthereumKVStoreInstance(conf.EthNode, conf.EthHexAddr, conf.EthHexKey)
				if err != nil {
					log.Println("Failed to NewEthereumKVStoreInstance", err)
					break
				}
				localconn = ethconn
			} else {
				ethconn, err := EthClientSDK.NewEthereumKVStoreInstance(shard.EthNode, shard.EthHexAddr, shard.EthHexKey)
				if err != nil {
					log.Println("Failed to NewEthereumKVStoreInstance", err)
					break
				}
				shards[shard.ID] = ethconn
			}

			confs[shard.ID] = shard
			log.Println("Sucess NewEthereumKVStoreInstance for shard ", shard.ID)
		case PARTITION_FAB().Shard:
			// #### disable Fabric sharding for ycsb tests ####
			// fabconn, err := FabClientSDK.NewFabricKVStoreInstance()
			// if err != nil {
			// 	log.Println("Failed to NewFabricKVStoreInstance", err)
			// 	break
			// }
			// shards[shard.ID] = fabconn
			// confs[shard.ID] = shard
			log.Println("Sucess NewFabricKVStoreInstance for shard ", shard.ID)
		default:
			log.Println("Error sharding key", shard.ID)
			break
		}
	}

	return &ShardingMgr{LocalShard: localconn, Shards: shards, Conf: confs, ShardNumber: conf.ShardNumber}, nil
}

func (mgr *ShardingMgr) partitionScheme(key string) string {

	partitionId := hash(key)%mgr.ShardNumber + 1

	return strconv.Itoa(partitionId)
}

func hash(data string) int {
	// 1. adler32
	sum := int(adler32.Checksum([]byte(data)))

	// 2. fnv
	// algorithm := fnv.New32a()
	// algorithm.Write([]byte(data))
	// sum := int(algorithm.Sum32())
	return sum
}

func partition(key string) string {
	if strings.HasPrefix(key, PARTITION_ETH().Key) {
		return PARTITION_ETH().Shard
	} else if strings.HasPrefix(key, PARTITION_FAB().Shard) {
		return PARTITION_FAB().Shard
	} else {
		return PARTITION_DEFAULT().Shard
	}
}

func (mgr *ShardingMgr) Read(ctx context.Context, key string) (string, error) {

	// switch partition(key) {
	// case PARTITION_ETH().Shard:
	// 	return mgr.EthConn.Read(key)

	// case PARTITION_FAB().Shard:
	// 	return mgr.FabConn.Read(key)

	// default:
	// 	return "", fmt.Errorf("Error sharding key %s", key)
	// }
	// partitionkey := mgr.partitionScheme(key)
	// log.Println("ShardingMgr read ", partitionkey)
	result, err := mgr.LocalShard.Read(ctx, key)
	if result == "" {
		for _, shard := range mgr.Shards {
			result, err = shard.Read(ctx, key)
			if result != "" {
				return result, err
			}
		}
	}

	return result, err

}

func (mgr *ShardingMgr) Write(ctx context.Context, key string, value string) (string, error) {
	// switch partition(key) {
	// case PARTITION_ETH().Shard:
	// 	return mgr.EthConn.Write(key, value)

	// case PARTITION_FAB().Shard:
	// 	return mgr.FabConn.Write(key, value)

	// default:
	// 	return fmt.Errorf("Error sharding key %s", key)
	// }
	// partitionkey := mgr.partitionScheme(key)
	// log.Println("ShardingMgr write ", partitionkey)
	return mgr.LocalShard.Write(ctx, key, value)
}

func (mgr *ShardingMgr) Verify(ctx context.Context, opt string, key string, tx string) (bool, error) {
	// switch partition(key) {
	// case PARTITION_ETH().Shard:
	// 	return mgr.EthConn.Write(key, value)

	// case PARTITION_FAB().Shard:
	// 	return mgr.FabConn.Write(key, value)

	// default:
	// 	return fmt.Errorf("Error sharding key %s", key)
	// }
	// partitionkey := mgr.partitionScheme(key)
	// log.Println("ShardingMgr verify ", partitionkey)
	result, err := mgr.LocalShard.Verify(ctx, opt, key, tx)
	if err != nil {
		for _, shard := range mgr.Shards {
			result, err = shard.Verify(ctx, opt, key, tx)
			if err == nil {
				return result, err
			}
		}
	}
	return result, err
}
