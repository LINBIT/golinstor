package client_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/LINBIT/golinstor/client"
	"github.com/LINBIT/golinstor/devicelayerkind"
)

func TestParse(t *testing.T) {
	no := false

	testcases := []struct {
		response string
		actual   interface{}
		expected interface{}
	}{
		{
			response: `{"name":"pvc-b5be6893-9892-4278-b2da-51a060fc4624","node_name":"demo1.linstor-days.at.linbit.com","props":{"StorPoolName":"thinpool"},"layer_object":{"children":[{"type":"STORAGE","storage":{"storage_volumes":[{"volume_number":0,"device_path":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":516096,"usable_size_kib":516096,"disk_state":"[]"}]}}],"type":"DRBD","drbd":{"drbd_resource_definition":{"peer_slots":7,"al_stripes":1,"al_stripe_size_kib":32,"port":7000,"transport_type":"IP","secret":"bNvYcSbPFPpbHZ9Gtq00","down":false},"node_id":0,"peer_slots":7,"al_stripes":1,"al_size":32,"drbd_volumes":[{"drbd_volume_definition":{"volume_number":0,"minor_number":1000},"device_path":"/dev/drbd1000","backing_device":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":512148,"usable_size_kib":512000}],"connections":{"demo2.linstor-days.at.linbit.com":{"connected":true,"message":"Connected"},"demo3.linstor-days.at.linbit.com":{"connected":false,"message":"Connecting"}},"promotion_score":10101,"may_promote":true}},"state":{"in_use":false},"uuid":"78f0d7fe-2b4d-4d5b-afb4-e1b1450c70cb","create_timestamp":1622636098831}`,
			actual:   &client.Resource{},
			expected: &client.Resource{
				Name:     "pvc-b5be6893-9892-4278-b2da-51a060fc4624",
				NodeName: "demo1.linstor-days.at.linbit.com",
				Props: map[string]string{
					"StorPoolName": "thinpool",
				},
				LayerObject: &client.ResourceLayer{
					Children: []client.ResourceLayer{
						{
							Type: devicelayerkind.Storage,
							Storage: &client.StorageResource{
								StorageVolumes: []client.StorageVolume{
									{
										VolumeNumber:     0,
										DevicePath:       "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
										AllocatedSizeKib: 516096,
										UsableSizeKib:    516096,
										DiskState:        "[]",
									},
								},
							},
						},
					},
					Type: devicelayerkind.Drbd,
					Drbd: &client.DrbdResource{
						DrbdResourceDefinition: client.DrbdResourceDefinitionLayer{
							PeerSlots:     7,
							AlStripes:     1,
							Port:          7000,
							TransportType: "IP",
							Secret:        "bNvYcSbPFPpbHZ9Gtq00",
						},
						DrbdVolumes: []client.DrbdVolume{
							{
								DrbdVolumeDefinition: client.DrbdVolumeDefinition{
									ResourceNameSuffix: "",
									VolumeNumber:       0,
									MinorNumber:        1000,
								},
								DevicePath:       "/dev/drbd1000",
								BackingDevice:    "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
								MetaDisk:         "",
								AllocatedSizeKib: 512148,
								UsableSizeKib:    512000,
								DiskState:        "",
								ExtMetaStorPool:  "",
							},
						},
						Connections: map[string]client.DrbdConnection{
							"demo2.linstor-days.at.linbit.com": {
								Connected: true,
								Message:   "Connected",
							},
							"demo3.linstor-days.at.linbit.com": {
								Connected: false,
								Message:   "Connecting",
							},
						},
						PeerSlots:      7,
						AlStripes:      1,
						AlSize:         32,
						PromotionScore: 10101,
						MayPromote:     true,
					},
				},
				State: &client.ResourceState{
					InUse: &no,
				},
				Uuid:            "78f0d7fe-2b4d-4d5b-afb4-e1b1450c70cb",
				CreateTimestamp: &client.TimeStampMs{Time: time.Unix(1622636098, 831_000_000)},
			},
		},
		{
			response: `{"name":"pvc-b5be6893-9892-4278-b2da-51a060fc4624","node_name":"demo1.linstor-days.at.linbit.com","props":{"StorPoolName":"thinpool"},"layer_object":{"children":[{"type":"STORAGE","storage":{"storage_volumes":[{"volume_number":0,"device_path":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":516096,"usable_size_kib":516096,"disk_state":"[]"}]}}],"type":"DRBD","drbd":{"drbd_resource_definition":{"peer_slots":7,"al_stripes":1,"al_stripe_size_kib":32,"port":7000,"transport_type":"IP","secret":"bNvYcSbPFPpbHZ9Gtq00","down":false},"node_id":0,"peer_slots":7,"al_stripes":1,"al_size":32,"drbd_volumes":[{"drbd_volume_definition":{"volume_number":0,"minor_number":1000},"device_path":"/dev/drbd1000","backing_device":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":512148,"usable_size_kib":512000}],"connections":{"demo2.linstor-days.at.linbit.com":{"connected":true,"message":"Connected"},"demo3.linstor-days.at.linbit.com":{"connected":false,"message":"Connecting"}},"promotion_score":10101,"may_promote":true}},"uuid":"78f0d7fe-2b4d-4d5b-afb4-e1b1450c70cb","create_timestamp":1622636098831,"volumes":[{"volume_number":0,"storage_pool_name":"thinpool","provider_kind":"LVM_THIN","device_path":"/dev/drbd1000","allocated_size_kib":206,"state":{"disk_state":"UpToDate"},"layer_data_list":[{"type":"DRBD","data":{"drbd_volume_definition":{"volume_number":0,"minor_number":1000},"device_path":"/dev/drbd1000","backing_device":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":512148,"usable_size_kib":512000}},{"type":"STORAGE","data":{"volume_number":0,"device_path":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":516096,"usable_size_kib":516096,"disk_state":"[]"}}],"uuid":"03b8ffd6-dbef-4745-87a0-46b4f8459e1e"}]}`,
			actual:   &client.ResourceWithVolumes{},
			expected: &client.ResourceWithVolumes{
				Resource: client.Resource{
					Name:     "pvc-b5be6893-9892-4278-b2da-51a060fc4624",
					NodeName: "demo1.linstor-days.at.linbit.com",
					Props: map[string]string{
						"StorPoolName": "thinpool",
					},
					LayerObject: &client.ResourceLayer{
						Children: []client.ResourceLayer{
							{
								Type: devicelayerkind.Storage,
								Storage: &client.StorageResource{
									StorageVolumes: []client.StorageVolume{
										{
											VolumeNumber:     0,
											DevicePath:       "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
											AllocatedSizeKib: 516096,
											UsableSizeKib:    516096,
											DiskState:        "[]",
										},
									},
								},
							},
						},
						Type: devicelayerkind.Drbd,
						Drbd: &client.DrbdResource{
							DrbdResourceDefinition: client.DrbdResourceDefinitionLayer{
								PeerSlots:     7,
								AlStripes:     1,
								Port:          7000,
								TransportType: "IP",
								Secret:        "bNvYcSbPFPpbHZ9Gtq00",
							},
							DrbdVolumes: []client.DrbdVolume{
								{
									DrbdVolumeDefinition: client.DrbdVolumeDefinition{
										ResourceNameSuffix: "",
										VolumeNumber:       0,
										MinorNumber:        1000,
									},
									DevicePath:       "/dev/drbd1000",
									BackingDevice:    "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
									MetaDisk:         "",
									AllocatedSizeKib: 512148,
									UsableSizeKib:    512000,
									DiskState:        "",
									ExtMetaStorPool:  "",
								},
							},
							Connections: map[string]client.DrbdConnection{
								"demo2.linstor-days.at.linbit.com": {
									Connected: true,
									Message:   "Connected",
								},
								"demo3.linstor-days.at.linbit.com": {
									Connected: false,
									Message:   "Connecting",
								},
							},
							PeerSlots:      7,
							AlStripes:      1,
							AlSize:         32,
							PromotionScore: 10101,
							MayPromote:     true,
						},
					},
					Uuid: "78f0d7fe-2b4d-4d5b-afb4-e1b1450c70cb",
				},
				CreateTimestamp: &client.TimeStampMs{Time: time.Unix(1622636098, 831_000_000)},
				Volumes: []client.Volume{
					{
						StoragePoolName:  "thinpool",
						ProviderKind:     client.LVM_THIN,
						DevicePath:       "/dev/drbd1000",
						AllocatedSizeKib: 206,
						State: client.VolumeState{
							DiskState: "UpToDate",
						},
						LayerDataList: []client.VolumeLayer{
							{
								Type: devicelayerkind.Drbd,
								Data: &client.DrbdVolume{
									DrbdVolumeDefinition: client.DrbdVolumeDefinition{
										MinorNumber: 1000,
									},
									DevicePath:       "/dev/drbd1000",
									BackingDevice:    "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
									AllocatedSizeKib: 512148,
									UsableSizeKib:    512000,
								},
							},
							{
								Type: devicelayerkind.Storage,
								Data: &client.StorageVolume{
									DevicePath:       "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
									AllocatedSizeKib: 516096,
									UsableSizeKib:    516096,
									DiskState:        "[]",
								},
							},
						},
						Uuid: "03b8ffd6-dbef-4745-87a0-46b4f8459e1e",
					},
				},
			},
		},
		{
			response: `{"volume_number":0,"storage_pool_name":"thinpool","provider_kind":"LVM_THIN","device_path":"/dev/drbd1000","allocated_size_kib":206,"state":{"disk_state":"UpToDate"},"layer_data_list":[{"type":"DRBD","data":{"drbd_volume_definition":{"volume_number":0,"minor_number":1000},"device_path":"/dev/drbd1000","backing_device":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":512148,"usable_size_kib":512000}},{"type":"STORAGE","data":{"volume_number":0,"device_path":"/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000","allocated_size_kib":516096,"usable_size_kib":516096,"disk_state":"[]"}}],"uuid":"03b8ffd6-dbef-4745-87a0-46b4f8459e1e"}`,
			actual:   &client.Volume{},
			expected: &client.Volume{
				StoragePoolName:  "thinpool",
				ProviderKind:     client.LVM_THIN,
				DevicePath:       "/dev/drbd1000",
				AllocatedSizeKib: 206,
				State: client.VolumeState{
					DiskState: "UpToDate",
				},
				LayerDataList: []client.VolumeLayer{
					{
						Type: devicelayerkind.Drbd,
						Data: &client.DrbdVolume{
							DrbdVolumeDefinition: client.DrbdVolumeDefinition{
								MinorNumber: 1000,
							},
							DevicePath:       "/dev/drbd1000",
							BackingDevice:    "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
							AllocatedSizeKib: 512148,
							UsableSizeKib:    512000,
						},
					},
					{
						Type: devicelayerkind.Storage,
						Data: &client.StorageVolume{
							DevicePath:       "/dev/linstor_thinpool/pvc-b5be6893-9892-4278-b2da-51a060fc4624_00000",
							AllocatedSizeKib: 516096,
							UsableSizeKib:    516096,
							DiskState:        "[]",
						},
					},
				},
				Uuid: "03b8ffd6-dbef-4745-87a0-46b4f8459e1e",
			},
		},
	}

	t.Parallel()
	for i := range testcases {
		tcase := &testcases[i]
		t.Run(reflect.TypeOf(tcase.expected).Name(), func(t *testing.T) {
			err := json.NewDecoder(strings.NewReader(tcase.response)).Decode(tcase.actual)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Equal(t, tcase.expected, tcase.actual)
		})
	}
}
