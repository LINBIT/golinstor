package devicelayerkind

type DeviceLayerKind string

const (
	Drbd       DeviceLayerKind = "drbd"
	Luks       DeviceLayerKind = "luks"
	Storage    DeviceLayerKind = "storage"
	Nvme       DeviceLayerKind = "nvme"
	Openflex   DeviceLayerKind = "openflex"
	Exos       DeviceLayerKind = "exos"
	Writecache DeviceLayerKind = "writecache"
	Cache      DeviceLayerKind = "cache"
)
