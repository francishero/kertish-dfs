package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/freakmaxi/2020-dfs/head-node/src/data"
	"github.com/freakmaxi/2020-dfs/head-node/src/manager"
	"github.com/freakmaxi/2020-dfs/head-node/src/routing"
	"github.com/freakmaxi/2020-dfs/head-node/src/services"
)

var version = "XX.X.XXXX"

func main() {
	args := os.Args[1:]
	if len(args) > 0 && strings.Compare(args[0], "--version") == 0 {
		fmt.Println(version)
		return
	}

	fmt.Printf("INFO: Starting 2020-dfs Head Node v%s\n", version)

	bindAddr := os.Getenv("BIND_ADDRESS")
	if len(bindAddr) == 0 {
		bindAddr = ":4000"
	}
	fmt.Printf("INFO: BIND_ADDRESS: %s\n", bindAddr)

	managerAddress := os.Getenv("MANAGER_ADDRESS")
	if len(managerAddress) == 0 {
		fmt.Println("ERROR: MANAGER_ADDRESS have to be specified")
		os.Exit(10)
	}
	fmt.Printf("INFO: MANAGER_ADDRESS: %s\n", managerAddress)

	mongoConn := os.Getenv("MONGO_CONN")
	if len(mongoConn) == 0 {
		fmt.Println("ERROR: MONGO_CONN have to be specified")
		os.Exit(11)
	}
	fmt.Printf("INFO: MONGO_CONN: %s\n", mongoConn)

	mongoDb := os.Getenv("MONGO_DATABASE")
	if len(mongoDb) == 0 {
		mongoDb = "2020-dfs"
	}
	fmt.Printf("INFO: MONGO_DATABASE: %s\n", mongoDb)

	redisConn := os.Getenv("REDIS_CONN")
	if len(redisConn) == 0 {
		fmt.Println("ERROR: REDIS_CONN have to be specified")
		os.Exit(12)
	}
	fmt.Printf("INFO: REDIS_CONN: %s\n", redisConn)

	mutex, err := data.NewMutex(redisConn)
	if err != nil {
		fmt.Printf("ERROR: Mutex Setup is failed. %s\n", err.Error())
		os.Exit(13)
	}

	conn, err := data.NewConnection(mongoConn)
	if err != nil {
		fmt.Printf("ERROR: MongoDB Connection is failed. %s\n", err.Error())
		os.Exit(15)
	}

	metadata, err := data.NewMetadata(mutex, conn, mongoDb)
	if err != nil {
		fmt.Printf("ERROR: Metadata Manager is failed. %s\n", err.Error())
		os.Exit(18)
	}

	cluster, err := manager.NewCluster([]string{managerAddress})
	if err != nil {
		fmt.Printf("ERROR: Cluster Manager is failed. %s\n", err.Error())
		os.Exit(20)
	}
	dfs := manager.NewDfs(metadata, cluster)
	dfsRouter := routing.NewDfsRouter(dfs)

	routerManager := routing.NewManager()
	routerManager.Add(dfsRouter)

	proxy := services.NewProxy(bindAddr, routerManager)
	proxy.Start()

	os.Exit(0)
}
