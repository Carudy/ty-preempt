package params

import "strconv"

type ChainConfig struct {
	ChainID                  int
	NodeID                   string
	ShardID                  string
	Shard_num                int
	Malicious_num            int    // per shard
	Path                     string // input file path
	Block_interval           int    // second
	MaxBlockSize             int
	MaxMigSize               int
	MaxMig2Size              int
	MaxMig1Size              int
	MaxAnnSize               int
	MaxCapSize               int
	Relay_interval           int
	MaxRelayBlockSize        int
	MinRelayBlockSize        int
	Inject_speed             int // tx count per second
	Max_Commit               int // 确认多少个交易后就迁移
	Max_Commit_Block         int // 确认多少个区块后就迁移，要×分片数量
	ClientSendTX             bool
	Stop_When_Migrating      bool //  迁移时是停止还是继续运行
	Lock_Acc_When_Migrating  bool //迁移时相关账户是否要锁住
	Bu_Tong_Bi_Li            bool //进行相关交易不同占比的实验, 地址：489338d5e8d42e8c923d1f47361d979503d4ad68
	Bu_Tong_Bi_Li_2          bool //进行相关交易不同占比的实验, 地址有10个
	Bu_Tong_Shi_Jian         bool //进行迁移请求不同时间被打包的实验, 地址：489338d5e8d42e8c923d1f47361d979503d4ad68
	Bu_Tong_Shi_Jian_Jian_Ge int  //进行迁移请求不同时间被打包的实验, 设置被打包的时间
	Fail                     bool //进行迁移迁移失败的实验, 地址：489338d5e8d42e8c923d1f47361d979503d4ad68
	Fail_Time                int  //进行迁移迁移失败的实验, 设置失败的时间
	Cross_Chain              bool //进行跨链迁移的实验, 地址：489338d5e8d42e8c923d1f47361d979503d4ad68
	Algorithm                bool
	Pressure                 bool   //压力测试，迁移多个账户
	PorC                     string //PageRank or CLPA
	MigrateBeforeInject      bool   //一开始就迁移
	OnlyOnce                 int    //1只迁移1次，否则设为1000000
	Not_Lock_immediately     bool
	RelayLock                bool
}

var (
	BaseIP     = "127.0.0.1"
	StartPort  = 32000
	ClientAddr = "127.0.0.1:" + strconv.Itoa(StartPort)

	// Configuration for auto-generation
	num_shards          = 2
	num_nodes_per_shard = 1

	// Auto-generated NodeTable
	NodeTable = func() map[string]map[string]string {
		nodeTable := make(map[string]map[string]string)
		for s := 0; s < num_shards; s++ {
			shardName := "S" + strconv.Itoa(s)
			nodeTable[shardName] = make(map[string]string)
			for n := 0; n < num_nodes_per_shard; n++ {
				nodeName := "N" + strconv.Itoa(n)
				port := StartPort + 100*(s+1) + n
				nodeTable[shardName][nodeName] = BaseIP + ":" + strconv.Itoa(port)
			}
		}
		return nodeTable
	}()

	// Auto-generated ShardTable
	ShardTable = func() map[string]int {
		shardTable := make(map[string]int)
		for s := 0; s < num_shards; s++ {
			shardName := "S" + strconv.Itoa(s)
			shardTable[shardName] = s
		}
		return shardTable
	}()

	// Auto-generated ShardTableInt2Str
	ShardTableInt2Str = func() map[int]string {
		shardTableInt2Str := make(map[int]string)
		for s := 0; s < num_shards; s++ {
			shardName := "S" + strconv.Itoa(s)
			shardTableInt2Str[s] = shardName
		}
		return shardTableInt2Str
	}()

	Config = &ChainConfig{
		ChainID:                  77,
		Block_interval:           6,
		MaxBlockSize:             2000,
		MaxMigSize:               1000,
		MaxMig2Size:              500,
		MaxMig1Size:              500,
		MaxAnnSize:               500,
		MaxCapSize:               500,
		MaxRelayBlockSize:        10,
		MinRelayBlockSize:        1,
		Inject_speed:             2000,
		Relay_interval:           1000,
		Max_Commit:               100000,
		Max_Commit_Block:         50,
		ClientSendTX:             true,
		Stop_When_Migrating:      false,
		Lock_Acc_When_Migrating:  false,
		Bu_Tong_Bi_Li:            false,
		Bu_Tong_Bi_Li_2:          true,
		Bu_Tong_Shi_Jian:         false,
		Bu_Tong_Shi_Jian_Jian_Ge: 3,
		Fail:                     false,
		Fail_Time:                8,
		Cross_Chain:              false,
		Algorithm:                false,
		Pressure:                 false,
		PorC:                     "CLPA",
		MigrateBeforeInject:      false,
		OnlyOnce:                 100,
		Not_Lock_immediately:     true,
		RelayLock:                false,
	}

	Init_addrs = []string{
		// "171382ed4571b1084bb5963053203c237dba6da9",
		// "2185bf3bfda43894efdcc1a3f4a99a7f160bc123",
		// "374be1a1d1ac0ff350dc9d0a0be3d059c7082791",
		// "42fd9ff72a798780c0dffc68f89b64ba240240dd",

		// "c418969d5f8948d9a40465f0a432d510b7e80b35",
		// "c418969d5f8948d9a40465f0a432d510b7e80b36",
		// "c418969d5f8948d9a40465f0a432d510b7e80b37",
		// "c418969d5f8948d9a40465f0a432d510b7e80b38",
		// "c418969d5f8948d9a40465f0a432d510b7e80b39",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3a",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3b",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3c",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3d",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3e",
		// "c418969d5f8948d9a40465f0a432d510b7e80b3f",
		// "72e5263ff33d2494692d7f94a758aa9f82062f73",
		// "108971c768fb92f287e76425086f9cfab9141b69",
		// "2b51be9c7b906180ff2d483ac1c3935149d8aa4e",
		// "2b51be9c7b906180ff2d483ac1c3935149d8aa4f",
		// "93175b6892a27d0fea4a451c32084d4e3d58275e",
		// "93175b6892a27d0fea4a451c32084d4e3d58275f",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b1c",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b1d",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b1e",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b1f",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b20",
		// "000f422887ea7d370ff31173fd3b46c8f66a5b21",
		// "298eebddf2a724f3423a424021ded398a9b0dc7d",
		// "9fc1e70e70e5a886945a3a8af437fc438fce599b",
		// "6758d7777813a335e90c94b867f3951d2e2e2cb0",
		// "4467470ed1b004fa21ebda7e2a3353ea0ef7ead6",
		// "bc22a279889f9ca5cf5b79ddc705292ba1f9c284",
		// "3dc8b4c9790d88b71aa0d002ba86f361af906990",
		// "76d8faff74268965d318064da8ca5918249c8d73",
		// "2faf487a4414fe77e2327f0bf4ae2a264a776ad2",
		// "25eaff5b179f209cf186b1cdcbfa463a69df4c45",
		// "aa024d398a3bfca7678f948067777ad63b2a14af",
		// "cdd37ada79f589c15bd4f8fd2083dc88e34a2af2",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276250",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276251",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276252",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276253",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276254",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276255",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276256",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276257",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276258",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f276259",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625a",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625b",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625c",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625d",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625e",
		// "89e8f6412bbd8cdae72693ba0e572c5b1f27625f",
		// "000000000000000000000000000000000000000f",
		// "0000000000000000000000000000000000000016",
	}
	Init_balance string = "10000000000000000000000000000000000000000" //40个0
	// Init_balance float64 = 200
)
