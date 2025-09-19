package devicelayerkind

type DeviceLayerKind string

const (
	Drbd       DeviceLayerKind = "DRBD"
	Luks       DeviceLayerKind = "LUKS"
	Storage    DeviceLayerKind = "STORAGE"
	Nvme       DeviceLayerKind = "NVME"
	Writecache DeviceLayerKind = "WRITECACHE"
	Cache      DeviceLayerKind = "CACHE"
	Bcache     DeviceLayerKind = "BCACHE"
)
