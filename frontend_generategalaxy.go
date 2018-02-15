package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
)

func generateGlobalGalaxyInfo(remoteIP string) bool {
	const fName = "(generateGlobalGalaxyInfo)"

	// generate xy coordinates
	var glblX int64
	var glblY int64
	for {
		// generating a uniformly random int64 value
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &glblX)
		// generating another uniformly random int64 value
		cryptorand.Read(b)
		buf = bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &glblY)
		// calculate hypotenuse and throw away any
		// coordinates that would fall outside the circle
		A2 := float64(glblX * glblX)
		B2 := float64(glblY * glblY)
		C := math.Sqrt(A2 + B2)
		if C >= galaxyRadius {
			continue
		}
		break
	}
	logDebug(fName, "generated possible globalX", fmt.Sprintf("%v", glblX))
	logDebug(fName, "generated possible globalY", fmt.Sprintf("%v", glblY))

	// generate name
	canidateName := generateName(GALAXYNAME, 0) // second arg is unused
	galName := canidateName.GetFullName()
	logDebug(fName, "generated possible galName", galName)

	// TODO //
	// ensure new coordinates are within range of prior galaxies
	// let everybody know about potential name/coordinates
	// if conflict arises, come up with different data and try again
	// other servers will need to be able to gracefully handle change

	GLOBAL.frontendData.Lock()
	GLOBAL.frontendData.GalGlobalX[galName] = glblX
	GLOBAL.frontendData.GalGlobalY[galName] = glblY
	GLOBAL.frontendData.GalClientIDs[galName] = []string{} // TODO // not sure about this
	GLOBAL.frontendData.GalBackends[galName] = []string{remoteIP}
	GLOBAL.frontendData.Unlock()

	// TODO //
	// handle the edge case where galaxyName already exists and/or
	// some other server comes up with a galaxy that conflicts
	// to do this we will need to wait 10 seconds and check again before returning

	return true
}
