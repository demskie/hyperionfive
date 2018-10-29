package subdivision

import (
	"log"
	"math"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/backendmessage"
	"github.com/gogo/protobuf/proto"
)

// VirtualizedGalaxy is used to subdivide the galaxy for efficient marshalling
type VirtualizedGalaxy struct {
	staticGalaxy        *backendmessage.StaticGalaxy
	staticGalaxyBytes   []byte
	dynamicSystemSlice  []*backendmessage.DynamicSystem
	dynamicSystemMap    map[int32]*backendmessage.DynamicSystem
	dynamicSectionSlice []*backendmessage.DynamicGalaxySection
	dynamicProtoBuffers []proto.Buffer
}

// NewVirtualizedGalaxy initializes the underlying datastructure
func NewVirtualizedGalaxy(args *arguments.Args, precision int32) *VirtualizedGalaxy {
	v := &VirtualizedGalaxy{}
	v.staticGalaxy = generateStaticGalaxy(args.BackendGalaxyID, precision, args.BackendGlobalX, args.BackendGlobalY)
	byteSlice, err := v.staticGalaxy.Marshal()
	if err != nil {
		log.Fatalln(err)
	}
	v.staticGalaxyBytes = byteSlice
	v.dynamicSystemSlice = make([]*backendmessage.DynamicSystem, 0, len(v.staticGalaxy.StaticSystem))
	v.dynamicSystemMap = map[int32]*backendmessage.DynamicSystem{}
	for sysID := range v.staticGalaxy.StaticSystem {
		ds := &backendmessage.DynamicSystem{}
		v.dynamicSystemSlice = append(v.dynamicSystemSlice, ds)
		v.dynamicSystemMap[sysID] = ds
	}
	v.dynamicSectionSlice = make([]*backendmessage.DynamicGalaxySection, precision*precision)
	for i := 0; i < len(v.dynamicSectionSlice); i++ {
		v.dynamicSectionSlice[i] = &backendmessage.DynamicGalaxySection{
			Id:            v.staticGalaxy.Id,
			Section:       int32(i),
			DynamicSystem: map[int32]*backendmessage.DynamicSystem{},
		}
	}
	for sysID, ss := range v.staticGalaxy.StaticSystem {
		i := calculateBucket(ss.InterX, ss.InterY, precision)
		v.dynamicSectionSlice[i].DynamicSystem[sysID] = v.dynamicSystemMap[sysID]
	}
	v.dynamicProtoBuffers = make([]proto.Buffer, precision*precision)
	return v
}

func (v *VirtualizedGalaxy) GetStaticGalaxy() *backendmessage.StaticGalaxy {
	return v.staticGalaxy
}

func (v *VirtualizedGalaxy) GetStaticGalaxyBytes() []byte {
	return v.staticGalaxyBytes
}

func (v *VirtualizedGalaxy) GetDynamicSystem(id int32) *backendmessage.DynamicSystem {
	return v.dynamicSystemMap[id]
}

func (v *VirtualizedGalaxy) GetAllDynamicSystems() []*backendmessage.DynamicSystem {
	return v.dynamicSystemSlice
}

func (v *VirtualizedGalaxy) ReserializeData() {
	for _, buf := range v.dynamicProtoBuffers {
		buf.Reset()
	}
}

func (v *VirtualizedGalaxy) GetDynamicGalaxySectionBytes(section int) []byte {
	buf := v.dynamicProtoBuffers[section]
	if len(buf.Bytes()) > 0 {
		return v.dynamicProtoBuffers[section].Bytes()
	}
	if cap(buf.Bytes()) > 2*len(buf.Bytes()) {
		buf.SetBuf(make([]byte, 0, len(buf.Bytes())))
	}
	err := v.dynamicProtoBuffers[section].Marshal(v.dynamicSectionSlice[section])
	if err != nil {
		log.Fatalln(err)
	}
	return v.dynamicProtoBuffers[section].Bytes()
}

func calculateBucket(x, y, diameter int32) int32 {
	xb, yb := diameter/2, diameter/2
	xb += x / (2 * (1 + (math.MaxInt32 / diameter)))
	yb += y / (2 * (1 + (math.MaxInt32 / diameter)))
	if xb > 0 {
		return yb + (xb * diameter)
	}
	return yb
}
