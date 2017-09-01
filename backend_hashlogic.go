package main

import (
	"encoding/binary"
	"hash/crc32"
)

func mergeSystemHashes() [16]map[string][4]byte {
	GLOBAL.backendData.RLock()
	result := [16]map[string][4]byte{}
	for i := range GLOBAL.backendData.SystemHashes {
		for neighbor := range GLOBAL.backendData.SystemHashes[i] {
			hash := crc32.NewIEEE()
			for _, sysHashArray := range GLOBAL.backendData.SystemHashes[i][neighbor] {
				sysHashSlice := []byte{}
				copy(sysHashSlice[:], sysHashArray[:])
				hash.Write(sysHashSlice)
			}
			hashSum := [4]byte{}
			copy(hashSum[:], hash.Sum(nil))
			result[i][neighbor] = hashSum
		}
	}
	GLOBAL.backendData.RUnlock()
	return result
}

func validateLocalHashes() bool {
	galaxyHashes := mergeSystemHashes()

	getUint := func(array [4]byte) uint64 {
		tmpSlice := []byte{}
		copy(tmpSlice, array[:])
		result, _ := binary.Uvarint(tmpSlice)
		return result
	}
	newestLocalUint := getUint(galaxyHashes[0]["0.0.0.0"])
	previousLocalUint := getUint(galaxyHashes[1]["0.0.0.0"])

	newestHashCount := map[uint64]int{}
	previousHashCount := map[uint64]int{}

	for neighbor := range galaxyHashes[0] {
		newestRemoteUint := getUint(galaxyHashes[0][neighbor])
		previousRemoteUint := getUint(galaxyHashes[1][neighbor])
		if neighbor == "0.0.0.0" {
			// representing ourselves in the tally
			newestHashCount[newestLocalUint]++
			previousHashCount[previousLocalUint]++
		} else if newestRemoteUint == newestLocalUint {
			// neighbor is perfectly in sync
			newestHashCount[newestRemoteUint]++
			previousHashCount[previousRemoteUint]++
		} else if newestRemoteUint == previousLocalUint {
			// neighbor is one tick behind us
			newestHashCount[newestLocalUint]++
			previousHashCount[newestRemoteUint]++
		} else if previousRemoteUint == newestLocalUint {
			// neighbor is one tick ahead of us
			newestHashCount[newestRemoteUint]++
			previousHashCount[previousLocalUint]++
		} else {
			// there is disagreement - will resolve later
			newestHashCount[newestRemoteUint]++
			previousHashCount[previousRemoteUint]++
		}
	}

	getMostCommonArray := func(hashCount map[uint64]int) uint64 {
		highestCount := 0
		var bestUint uint64
		bestUint = 0
		for canidateUint, count := range hashCount {
			if (count == highestCount) && (canidateUint > bestUint) {
				bestUint = canidateUint
			} else if count > highestCount {
				highestCount = count
				bestUint = canidateUint
			}
		}
		return bestUint
	}
	newestPopularUint := getMostCommonArray(newestHashCount)
	previousPopularUint := getMostCommonArray(previousHashCount)

	if newestLocalUint == newestPopularUint {
		logDebug("WE ARE INSYNC WITH MOST OF THE CLUSTER")
		return true
	} else if newestLocalUint == previousPopularUint {
		logDebug("WE ARE ONE TICK AHEAD MOST OF THE CLUSTER")
		return true
	} else if previousLocalUint == previousPopularUint {
		logDebug("WE JUST DESYNCED WITH THE REST OF THE CLUSTER")
		return false
	}

	return false
}
