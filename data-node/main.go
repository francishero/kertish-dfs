package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/freakmaxi/kertish-dfs/data-node/cache"
	"github.com/freakmaxi/kertish-dfs/data-node/filesystem"
	"github.com/freakmaxi/kertish-dfs/data-node/manager"
	"github.com/freakmaxi/kertish-dfs/data-node/service"
)

var version = "XX.X.XXXX"

func main() {
	printWelcome()

	args := os.Args[1:]
	if len(args) > 0 && strings.Compare(args[0], "--version") == 0 {
		fmt.Println(version)
		return
	}

	fmt.Println("INFO: ------------ Starting Data Node ------------")

	hardwareAddr, err := findHardwareAddress()
	if err != nil {
		fmt.Printf("ERROR: Unable to read hardware details: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("INFO: HARDWARE_ID: %s\n", hardwareAddr)

	bindAddr := os.Getenv("BIND_ADDRESS")
	if matched, err := regexp.MatchString(`:\d{1,5}$`, bindAddr); err != nil || !matched {
		bindAddr = fmt.Sprintf("%s:9430", bindAddr)
	}
	fmt.Printf("INFO: BIND_ADDRESS: %s\n", bindAddr)

	managerAddress := os.Getenv("MANAGER_ADDRESS")
	if len(managerAddress) == 0 {
		fmt.Println("ERROR: MANAGER_ADDRESS have to be specified")
		os.Exit(10)
	}
	fmt.Printf("INFO: MANAGER_ADDRESS: %s\n", managerAddress)

	sizeString := os.Getenv("SIZE")
	if len(sizeString) == 0 {
		fmt.Println("ERROR: SIZE have to be specified")
		os.Exit(50)
	}
	size, err := strconv.ParseUint(sizeString, 10, 64)
	if err != nil {
		fmt.Printf("ERROR: File System size is wrong: %s\n", err.Error())
		os.Exit(51)
	}
	if size == 0 {
		fmt.Println("ERROR: File System size can not be 0")
		os.Exit(52)
	}
	fmt.Printf("INFO: SIZE: %s (%s Gb)\n", sizeString, strconv.FormatUint(size/(1024*1024*1024), 10))

	rootPath := os.Getenv("ROOT_PATH")
	if len(rootPath) == 0 {
		rootPath = "/opt"
	}
	fmt.Printf("INFO: ROOT_PATH: %s\n", rootPath)

	fs := filesystem.NewManager(rootPath, size)
	n, err := manager.NewNode(strings.Split(managerAddress, ","))
	if err != nil {
		fmt.Printf("ERROR: Node Manager creation is failed: %s\n", err.Error())
		os.Exit(100)
	}

	cacheLifetime := 360
	cacheLimitString := os.Getenv("CACHE_LIMIT")
	if len(cacheLimitString) == 0 {
		cacheLimitString = "0"
	}
	cacheLimit, err := strconv.ParseUint(cacheLimitString, 10, 64)
	if err != nil {
		fmt.Printf("ERROR: Cache Limit size is wrong: %s\n", err.Error())
		os.Exit(120)
	}
	if cacheLimit == 0 {
		fmt.Println("WARN: Cache is disabled")
	} else {
		fmt.Printf("INFO: CACHE_LIMIT: %s (%s Gb)\n", cacheLimitString, strconv.FormatUint(cacheLimit/(1024*1024*1024), 10))

		ccLifetimeString := os.Getenv("CACHE_LIFETIME")
		if len(ccLifetimeString) == 0 {
			ccLifetimeString = "360"
		}
		ccLifetime, err := strconv.ParseUint(ccLifetimeString, 10, 64)
		if err != nil {
			fmt.Printf("ERROR: Cache Lifetime is wrong: %s\n", err.Error())
			os.Exit(130)
		}
		if ccLifetime == 0 {
			fmt.Println("ERROR: Cache Lifetime can not be 0")
			os.Exit(131)
		}
		fmt.Printf("INFO: CACHE_LIFETIME: %s min.\n", ccLifetimeString)
	}

	cc := cache.NewContainer(cacheLimit, time.Minute*time.Duration(cacheLifetime))

	c, err := service.NewCommander(fs, cc, n, hardwareAddr)
	if err != nil {
		fmt.Printf("ERROR: Commander creation is failed: %s\n", err.Error())
		os.Exit(200)
	}

	s, err := service.NewServer(bindAddr, c)
	if err != nil {
		fmt.Printf("ERROR: Server creation is failed: %s\n", err.Error())
		os.Exit(300)
	}

	fmt.Print("INFO: Waiting for handshake...")
	if err := n.Handshake(hardwareAddr, bindAddr, size); err != nil {
		fmt.Printf(" %s\n", err.Error())
		fmt.Printf("INFO: Data Node is starting as stand-alone on %s\n", bindAddr)
	} else {
		fmt.Print(" done.\n")

		mode := "MASTER"
		if len(n.MasterAddress()) > 0 {
			mode = "SLAVE"

			go func() {
				if err := fs.Sync(n.MasterAddress()); err != nil {
					fmt.Printf("WARN: Sync is failed (%s): %s\n", n.MasterAddress(), err.Error())
				}
			}()
		}
		fmt.Printf("INFO: Data Node (%s) in Cluster (%s) is starting on %s as %s\n", n.NodeId(), n.ClusterId(), bindAddr, mode)
	}
	if err := s.Listen(); err != nil {
		fmt.Printf("ERROR: Server listening is failed: %s\n", err.Error())
		os.Exit(400)
	}

	os.Exit(0)
}

func findHardwareAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, in := range interfaces {
		addrs, err := in.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			switch addr.(type) {
			case *net.IPNet:
				addrIp := addr.(*net.IPNet).IP

				if addrIp.To4() == nil || addrIp.IsLoopback() {
					continue
				}

				return in.HardwareAddr.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no suitable hardware address is found")
}

func printWelcome() {
	fmt.Println()
	fmt.Println("     'o@@@@@@o,  o@@@@@@o")
	fmt.Println("   'o@@@@o/-\\@@|@@/--\\@@@o             __ _  ____  ____  ____  __  ____  _  _")
	fmt.Println("  `o@/.       `@@~      o@@o          (  / )(  __)(  _ \\(_  _)(  )/ ___)/ )( \\")
	fmt.Println("  o@@:   oo    @@ .@@@. :@@~           )  (  ) _)  )   /  )(   )( \\___ \\) __ (")
	fmt.Println("  o@@,  .@@@.  @@=  oo  o@o`          (__\\_)(____)(__\\_) (__) (__)(____/\\_)(_/")
	fmt.Println("  '@@%`      `@@@@o....@@%`                                  ____  ____  ____")
	fmt.Println("   :@@@@o....@@@@@@@@@@@@@%~                                (    \\(  __)/ ___)")
	fmt.Println(" .oo@@@@@@@@@@@@@@@@@@@@@@@@o~`    .@@@@`                    ) D ( ) _) \\___ \\")
	fmt.Println("o@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@o . @@@oo@@@@@               (____/(__)  (____/")
	fmt.Printf("o@@@@@@%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@`  @o  @               version %s\n", version)
	fmt.Println("o@@@@@@:~O@@@@@@@@@@@@@@@@@@@@@@@@@@@ooo@@@@@")
	fmt.Println(" ~o@@@@|  `O@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@/`")
	fmt.Println("   `~=/`   *O@@@@@@@@@@@@@@@@@@@@@@@@@@@O/")
	fmt.Println("              \\\\O@@@@@@@@@@@@@@@@@@@@@O/`")
	fmt.Println("                 `\\\\|O@@@@@@@@@0oo/:")
	fmt.Println()
	fmt.Printf("Visit: https://github.com/freakmaxi/kertish-dfs\n")
	fmt.Println()
}