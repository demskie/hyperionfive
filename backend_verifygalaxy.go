package main

import (
	"fmt"
	"sort"
	"time"
)

func verifyGalaxy(galaxy *Galaxy) {
	const fName = "(fixBadSystemData)"

	start := time.Now()
	defer logTelemetry(fName, fmt.Sprintf("finished in %v milliseconds",
		time.Since(start).Nanoseconds()/1e6))

	GLOBAL.backendData.Lock()
	defer GLOBAL.backendData.Unlock()

	// regenerate systemsInRecovery based on what hasn't been fixed
	for sysName, localHash := range GLOBAL.backendData.SystemHashes {
		type data struct {
			count       int
			neighborIPs []string
		}
		popularity := map[int64]*data{}
		for neighborIP := range GLOBAL.backendData.NeighborSystemHashes {
			remoteHash := GLOBAL.backendData.NeighborSystemHashes[neighborIP][sysName]
			_, ok := popularity[remoteHash]
			if ok {
				popularity[remoteHash].count++
				popularity[remoteHash].neighborIPs = append([]string{}, neighborIP)
			} else {
				popularity[remoteHash] = &data{
					count:       1,
					neighborIPs: []string{neighborIP},
				}
			}
		}
		highestCount := 0
		bestHash := int64(0)
		for someHash, someData := range popularity {
			if someData.count < highestCount {
				continue
			}
			highestCount = someData.count
			someData.neighborIPs = append([]string{}, getLocalAddress().IP.String())
			sort.Strings(someData.neighborIPs)
			lowestAlphaIP := someData.neighborIPs[0]
			if someData.count == highestCount {
				if lowestAlphaIP < getLocalAddress().IP.String() {
					bestHash = someHash
				}
			} else {
				bestHash = someHash
			}
		}
		if localHash != bestHash {
			// false means systemData hasn't been received yet
			// whereas true means systemData is ready
			GLOBAL.backendData.SystemsInRecovery[sysName] = false
		}
	}

	// for the systems that need recovery we are going to
	// divide the responsibility evenly to the various neighbors
	GLOBAL.backendData.NeighborRecoveryMapping = map[string][]string{}
	if len(GLOBAL.backendData.Backends) != 0 {
		rnum := getCryptoRnum()
		for sysName := range GLOBAL.backendData.SystemsInRecovery {
			i := rnum.Intn(len(GLOBAL.backendData.Backends))
			neighbor := GLOBAL.backendData.Backends[i]
			_, ok := GLOBAL.backendData.NeighborRecoveryMapping[neighbor]
			if ok {
				GLOBAL.backendData.NeighborRecoveryMapping[neighbor] = []string{sysName}
			} else {
				GLOBAL.backendData.NeighborRecoveryMapping[neighbor] = append(
					GLOBAL.backendData.NeighborRecoveryMapping[neighbor], sysName)
			}
		}
	}
}
