package devicelayerkind

type DeviceLayerKind string

const (
	Drbd       DeviceLayerKind = "DRBD"
	Luks       DeviceLayerKind = "LUKS"
	Storage    DeviceLayerKind = "STORAGE"
	Nvme       DeviceLayerKind = "NVME"
	Exos       DeviceLayerKind = "EXOS"
	Writecache DeviceLayerKind = "WRITECACHE"
	Cache      DeviceLayerKind = "CACHE"
	Bcache     DeviceLayerKind = "BCACHE"
)
