package functions

import (
	"flag"
	"fmt"
)

const (
	DB_TYPE_DESKTOP    = "desktop"
	DB_TYPE_DW         = "dw"
	DB_TYPE_MIXED      = "mixed"
	DB_TYPE_OLTP       = "oltp"
	DB_TYPE_WEB        = "web"
	DEFAULT_DB_VERSION = "15"
	HARD_DRIVE_HDD     = "hdd"
	HARD_DRIVE_SAN     = "san"
	HARD_DRIVE_SSD     = "ssd"
	SIZE_UNIT_GB       = "GB"
	SIZE_UNIT_MB       = "MB"
)

var (
	checkpointCompletionTarget float32 = 0.9
	defaultStatisticsTarget    int
	effectiveCacheSize         int
	effectiveIoConcurrency     int
	finalConnectionNum         int
	hugePages                  string
	maintenanceWorkMem         int
	maxWalSize                 int
	minWalSize                 int
	randomPageCost             float32
	sharedBuffers              int
	walBuffersValue            int
	workMemBase                int
	workMemResult              int
	workMemValue               int
)

func byteSize(size int) string {
	var result int
	if size%1024 != 0 || size < 1024 {
		return fmt.Sprintf("%dKB", size)
	}
	result = size / 1024
	if result%1024 != 0 || result < 1024 {
		return fmt.Sprintf("%dMB", result)
	}
	result = result / 1024
	if result%1024 != 0 || result < 1024 {
		return fmt.Sprintf("%dGB", result)
	}
	return fmt.Sprintf("%d", size)
}

func PgTune() {
	connectionNum := flag.Int("connections", 0, "Maximum number of PostgreSQL client connections")
	cpuNum := flag.Int("cpus", 0, "Number of CPUs, which PostgreSQL can leverage\nCPUs = threads per core * cores per socket * sockets")
	dbType := flag.String("db-type", DB_TYPE_WEB, "What type of application PostgreSQL is installed for")
	dbVersion := flag.String("db-version", DEFAULT_DB_VERSION, "PostgreSQL version - can be found out via 'SELECT version();'")
	hdType := flag.String("hd-type", HARD_DRIVE_SSD, "Type of data storage device")
	totalMemory := flag.Int("total-memory", 0, "How much memory can PostgreSQL take advantage of")
	totalMemoryUnit := flag.String("total-memory-unit", SIZE_UNIT_GB, "Memory unit")
	flag.Parse()

	if *connectionNum > 0 {
		fmt.Println("# Connection num:", *connectionNum)
	}

	SIZE_UNIT_MAP := map[string]int{
		"KB": 1024,
		"MB": 1048576,
		"GB": 1073741824,
		"TB": 1099511627776,
	}

	totalMemoryInBytes := *totalMemory * SIZE_UNIT_MAP[*totalMemoryUnit]
	totalMemoryInKb := totalMemoryInBytes / SIZE_UNIT_MAP["KB"]

	DEFAULT_STATISTICS_TARGET_MAP := map[string]int{
		DB_TYPE_DESKTOP: 100,
		DB_TYPE_DW:      500,
		DB_TYPE_MIXED:   100,
		DB_TYPE_OLTP:    100,
		DB_TYPE_WEB:     100,
	}
	defaultStatisticsTarget = DEFAULT_STATISTICS_TARGET_MAP[*dbType]

	EFFECTIVE_CACHE_SIZE_MAP := map[string]int{
		DB_TYPE_DESKTOP: totalMemoryInKb / 4,
		DB_TYPE_DW:      (totalMemoryInKb * 3) / 4,
		DB_TYPE_MIXED:   (totalMemoryInKb * 3) / 4,
		DB_TYPE_OLTP:    (totalMemoryInKb * 3) / 4,
		DB_TYPE_WEB:     (totalMemoryInKb * 3) / 4,
	}
	effectiveCacheSize = EFFECTIVE_CACHE_SIZE_MAP[*dbType]

	EFFECTIVE_IO_CONCURRENCY := map[string]int{
		HARD_DRIVE_HDD: 2,
		HARD_DRIVE_SAN: 300,
		HARD_DRIVE_SSD: 200,
	}
	effectiveIoConcurrency = EFFECTIVE_IO_CONCURRENCY[*hdType]

	if *connectionNum < 1 {
		CONNECTION_NUM_MAP := map[string]int{
			DB_TYPE_DESKTOP: 20,
			DB_TYPE_DW:      40,
			DB_TYPE_MIXED:   100,
			DB_TYPE_OLTP:    300,
			DB_TYPE_WEB:     200,
		}
		finalConnectionNum = CONNECTION_NUM_MAP[*dbType]
	} else {
		finalConnectionNum = *connectionNum
	}

	if totalMemoryInKb >= 33554432 {
		hugePages = "try"
	} else {
		hugePages = "off"
	}

	MAINTENANCE_WORK_MEM_MAP := map[string]int{
		DB_TYPE_DESKTOP: totalMemoryInKb / 16,
		DB_TYPE_DW:      totalMemoryInKb / 8,
		DB_TYPE_MIXED:   totalMemoryInKb / 16,
		DB_TYPE_OLTP:    totalMemoryInKb / 16,
		DB_TYPE_WEB:     totalMemoryInKb / 16,
	}
	maintenanceWorkMem = MAINTENANCE_WORK_MEM_MAP[*dbType]
	// Cap maintenance RAM at 2 GB on servers with lots of memory
	memoryLimit := (SIZE_UNIT_MAP["GB"] * 2) / SIZE_UNIT_MAP["KB"]
	if maintenanceWorkMem > memoryLimit {
		maintenanceWorkMem = memoryLimit
	}

	MIN_WAL_SIZE_MAP := map[string]int{
		DB_TYPE_DESKTOP: (SIZE_UNIT_MAP["MB"] * 100) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_DW:      (SIZE_UNIT_MAP["MB"] * 4096) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_MIXED:   (SIZE_UNIT_MAP["MB"] * 1024) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_OLTP:    (SIZE_UNIT_MAP["MB"] * 2048) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_WEB:     (SIZE_UNIT_MAP["MB"] * 1024) / SIZE_UNIT_MAP["KB"],
	}
	minWalSize = MIN_WAL_SIZE_MAP[*dbType]

	MAX_WAL_SIZE_MAP := map[string]int{
		DB_TYPE_DESKTOP: (SIZE_UNIT_MAP["MB"] * 2048) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_DW:      (SIZE_UNIT_MAP["MB"] * 16384) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_MIXED:   (SIZE_UNIT_MAP["MB"] * 4096) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_OLTP:    (SIZE_UNIT_MAP["MB"] * 8192) / SIZE_UNIT_MAP["KB"],
		DB_TYPE_WEB:     (SIZE_UNIT_MAP["MB"] * 4096) / SIZE_UNIT_MAP["KB"],
	}
	maxWalSize = MAX_WAL_SIZE_MAP[*dbType]

	RANDOM_PAGE_COST_MAP := map[string]float32{
		HARD_DRIVE_HDD: 4,
		HARD_DRIVE_SAN: 1.1,
		HARD_DRIVE_SSD: 1.1,
	}
	randomPageCost = RANDOM_PAGE_COST_MAP[*hdType]

	SHARED_BUFFERS_MAP := map[string]int{
		DB_TYPE_DESKTOP: totalMemoryInKb / 16,
		DB_TYPE_DW:      totalMemoryInKb / 4,
		DB_TYPE_MIXED:   totalMemoryInKb / 4,
		DB_TYPE_OLTP:    totalMemoryInKb / 4,
		DB_TYPE_WEB:     totalMemoryInKb / 4,
	}
	sharedBuffers = SHARED_BUFFERS_MAP[*dbType]

	/* 
	Follow auto-tuning guideline for 'wal_buffers' added in 9.1, where it's
	set to 3% of 'shared_buffers' up to a maximum of 16 MB.
	*/
	walBuffersValue = (sharedBuffers * 3) / 100
	maxWalBuffer := (SIZE_UNIT_MAP["MB"] * 16) / SIZE_UNIT_MAP["KB"]
	if walBuffersValue > maxWalBuffer {
		walBuffersValue = maxWalBuffer
	}
	walBufferNearValue := (SIZE_UNIT_MAP["MB"] * 14) / SIZE_UNIT_MAP["KB"]
	if walBuffersValue > walBufferNearValue && walBuffersValue < maxWalBuffer {
		walBuffersValue = maxWalBuffer
	}
	// If less than 32 KB then set it to minimum
	if walBuffersValue < 32 {
		walBuffersValue = 32
	}

	if *cpuNum >= 2 {
		workMemBase = *cpuNum / 2
	} else {
		workMemBase = 1
	}

	/*
	'work_mem' is assigned any time a query calls for a sort, or a hash,
	or any other structure that needs a space allocation, which can happen
	multiple times per query. So you're better off assuming
	max_connections * 2 or max_connections * 3 is the amount of RAM that
	will actually be leveraged. At the very least, you need to subtract
	'shared_buffers' from the amount you distribute to connections
	in 'work_mem'. The other stuff to consider is that there's no reason
	to run on the edge of available memory. If you carry out that, there's
	a high risk the out-of-memory killer will come along and start killing
	PostgreSQL backends. Always leave a buffer of some kind in case of
	spikes in memory usage. So your maximum amount of memory available
	in 'work_mem' should be
	((RAM - shared_buffers) / (max_connections * 3)) /
	max_parallel_workers_per_gather
	*/
	workMemValue = ((totalMemoryInKb - sharedBuffers) / (finalConnectionNum * 3)) / workMemBase
	WORK_MEM_MAP := map[string]int{
		DB_TYPE_DESKTOP: workMemValue / 6,
		DB_TYPE_DW:      workMemValue / 2,
		DB_TYPE_MIXED:   workMemValue / 2,
		DB_TYPE_OLTP:    workMemValue,
		DB_TYPE_WEB:     workMemValue,
	}
	workMemResult = WORK_MEM_MAP[*dbType]
	// If less than 64 KB then set it to minimum
	if workMemResult < 64 {
		workMemResult = 64
	}

	fmt.Println("# CPUs num:", *cpuNum)
	fmt.Println("# Data Storage:", *hdType)
	fmt.Println("# DB Type:", *dbType)
	fmt.Println("# DB Version:", *dbVersion)
	fmt.Println("# OS Type:", "Linux")
	fmt.Println("# Total Memory (RAM):", *totalMemory, *totalMemoryUnit)
	fmt.Println("")
	fmt.Println("checkpoint_completion_target", "=", checkpointCompletionTarget)
	fmt.Println("default_statistics_target", "=", defaultStatisticsTarget)
	fmt.Println("effective_cache_size", "=", byteSize(effectiveCacheSize))
	fmt.Println("effective_io_concurrency", "=", effectiveIoConcurrency)
	fmt.Println("huge_pages", "=", hugePages)
	fmt.Println("maintenance_work_mem", "=", byteSize(maintenanceWorkMem))
	fmt.Println("max_connections", "=", finalConnectionNum)
	fmt.Println("max_parallel_workers", "=", *cpuNum)
	fmt.Println("max_wal_size", "=", byteSize(maxWalSize))
	fmt.Println("min_wal_size", "=", byteSize(minWalSize))
	fmt.Println("random_page_cost", "=", randomPageCost)
	fmt.Println("shared_buffers", "=", byteSize(sharedBuffers))
	fmt.Println("wal_buffers", "=", byteSize(walBuffersValue))
	fmt.Println("work_mem", "=", byteSize(workMemResult))
}